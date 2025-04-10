package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

// Global variable to store the grammar checker instance
var grammarChecker *GrammarChecker

// GrammarChecker provides grammar checking capabilities
type GrammarChecker struct {
	serverURL  string
	serverProc *exec.Cmd
}

// LanguageToolResult represents the response from LanguageTool
type LanguageToolResult struct {
	Software struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"software"`
	Matches []struct {
		Message      string        `json:"message"`
		Offset       int           `json:"offset"`
		Length       int           `json:"length"`
		Replacements []Replacement `json:"replacements"`
		Context      struct {
			Text   string `json:"text"`
			Offset int    `json:"offset"`
			Length int    `json:"length"`
		} `json:"context"`
		Rule struct {
			ID          string `json:"id"`
			Description string `json:"description"`
			Category    struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"category"`
		} `json:"rule"`
	} `json:"matches"`
}

// NewGrammarChecker creates a new grammar checker
func NewGrammarChecker() (*GrammarChecker, error) {
	gc := &GrammarChecker{
		serverURL: "http://localhost:8081/v2/check",
	}

	// Start the LanguageTool server
	err := gc.StartServer()
	if err != nil {
		return nil, fmt.Errorf("failed to start LanguageTool server: %w", err)
	}

	// Wait for the server to start
	time.Sleep(5 * time.Second)

	return gc, nil
}

// StartServer starts the LanguageTool server
func (gc *GrammarChecker) StartServer() error {
	gc.serverProc = exec.Command("java", "-cp", "external/LanguageTool-6.6/languagetool-server.jar", "org.languagetool.server.HTTPServer", "--port", "8081")

	err := gc.serverProc.Start()
	if err != nil {
		return err
	}

	fmt.Println("LanguageTool server started on port 8081")
	return nil
}

// StopServer stops the LanguageTool server
func (gc *GrammarChecker) StopServer() error {
	if gc.serverProc != nil && gc.serverProc.Process != nil {
		return gc.serverProc.Process.Kill()
	}
	return nil
}

// CheckText checks the grammar of the given text
func (gc *GrammarChecker) CheckText(text string, lang string) (*LanguageToolResult, error) {
	if lang == "" {
		lang = "en-US" // Default to English
	}

	params := url.Values{}
	params.Add("text", text)
	params.Add("language", lang)

	resp, err := http.PostForm(gc.serverURL, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result LanguageToolResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// FormatCorrections formats grammar correction suggestions
func (gc *GrammarChecker) FormatCorrections(result *LanguageToolResult) string {
	if len(result.Matches) == 0 {
		return "No grammar issues found."
	}

	var builder strings.Builder
	builder.WriteString("Grammar suggestions:\n\n")

	for i, match := range result.Matches {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, match.Message))
		builder.WriteString(fmt.Sprintf("   Context: %s\n", match.Context.Text))

		if len(match.Replacements) > 0 {
			builder.WriteString("   Suggestions: ")
			// Extract replacement values and join them
			suggestionValues := make([]string, 0, min(3, len(match.Replacements)))
			for j, repl := range match.Replacements {
				if j >= 3 {
					break // Limit to 3 suggestions
				}
				suggestionValues = append(suggestionValues, repl.Value)
			}
			suggestions := strings.Join(suggestionValues, ", ")
			builder.WriteString(suggestions)
			builder.WriteString("\n")
		}

		builder.WriteString("\n")
	}

	return builder.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// InitGrammarChecker initializes the grammar checker
func InitGrammarChecker() {
	var err error
	grammarChecker, err = NewGrammarChecker()
	if err != nil {
		log.Printf("Warning: Grammar checker initialization failed: %v", err)
		// Continue without the grammar checker
	}
}

// CheckGrammar checks the grammar of the given text
func CheckGrammar(text string) ([]GrammarIssue, error) {
	if grammarChecker == nil {
		return nil, fmt.Errorf("grammar checker not initialized")
	}

	result, err := grammarChecker.CheckText(text, "")
	if err != nil {
		return nil, err
	}

	// Convert LanguageTool results to GrammarIssue
	issues := make([]GrammarIssue, len(result.Matches))
	for i, match := range result.Matches {
		// Extract replacement values from the Replacement objects
		suggestions := make([]string, len(match.Replacements))
		for j, repl := range match.Replacements {
			suggestions[j] = repl.Value
		}

		issues[i] = GrammarIssue{
			Message:     match.Message,
			Context:     match.Context.Text,
			Offset:      match.Offset,
			Length:      match.Length,
			Suggestions: suggestions,
		}
	}

	return issues, nil
}
