package main

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GrammarIssue defines the structure for a single grammar problem
type GrammarIssue struct {
	Message     string   `json:"message"`     // Description of the issue
	Context     string   `json:"context"`     // Text snippet where issue occurs
	Offset      int      `json:"offset"`      // Character offset in the original text
	Length      int      `json:"length"`      // Length of the problematic text span
	Suggestions []string `json:"suggestions"` // Suggested replacements
}

// Note defines the structure for a note stored in MongoDB
type Note struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"` // MongoDB object ID
	OriginalFilename string             `bson:"originalFilename"`
	MarkdownContent  string             `bson:"markdownContent"`
	HTMLContent      string             `bson:"htmlContent"`
	GrammarIssues    []GrammarIssue     `bson:"grammarIssues"`
	CreatedAt        time.Time          `bson:"createdAt"`
}

// --- LanguageTool JSON Output Structures ---
// These match the JSON output from languagetool --json flag

type LTMatch struct {
	Message      string        `json:"message"`
	ShortMessage string        `json:"shortMessage"`
	Replacements []Replacement `json:"replacements"`
	Offset       int           `json:"offset"`
	Length       int           `json:"length"`
	Context      Context       `json:"context"`
	Sentence     string        `json:"sentence"`
	Type         TypeInfo      `json:"type"`
	Rule         RuleInfo      `json:"rule"`
	// Add other fields if needed (ignoreUnknown, contextForSureMatch, etc.)
}

type Replacement struct {
	Value string `json:"value"`
}

type Context struct {
	Text   string `json:"text"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
}

type TypeInfo struct {
	TypeName string `json:"typeName"`
}

type RuleInfo struct {
	ID          string       `json:"id"`
	Description string       `json:"description"`
	IssueType   string       `json:"issueType"`
	Category    CategoryInfo `json:"category"`
	// IsPremium bool `json:"isPremium"` // Uncomment if using premium rules
}

type CategoryInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type LTResponse struct {
	Software SoftwareInfo `json:"software"`
	Warnings WarningsInfo `json:"warnings"`
	Language LanguageInfo `json:"language"`
	Matches  []LTMatch    `json:"matches"`
}

type SoftwareInfo struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	BuildDate  string `json:"buildDate"`
	APIVersion int    `json:"apiVersion"`
	Premium    bool   `json:"premium"` // Indicates if premium features are available
	Status     string `json:"status"`
}

type WarningsInfo struct {
	IncompleteResults bool `json:"incompleteResults"`
}

type LanguageInfo struct {
	Name         string               `json:"name"`
	Code         string               `json:"code"`
	DetectedLang DetectedLanguageInfo `json:"detectedLanguage"`
}
type DetectedLanguageInfo struct {
	Name       string  `json:"name"`
	Code       string  `json:"code"`
	Confidence float64 `json:"confidence"`
	Source     string  `json:"source"` // e.g., "ngram" or "user"
}
