package main

import (
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	md "github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"go.mongodb.org/mongo-driver/mongo"
)

var templates *template.Template

// LoadTemplates parses all HTML templates
func LoadTemplates() {
	// Define template functions
	funcmap := template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s) // Marks the string as safe HTML content
		},
	}

	// Use Funcs to add custom template functions
	templates = template.Must(template.New("").Funcs(funcmap).ParseGlob("templates/*.html"))
	log.Println("HTML Templates loaded successfully.")
}

// renderTemplate executes a specific template with data
func renderTemplate(w http.ResponseWriter, tmplName string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmplName, data)
	if err != nil {
		log.Printf("Error executing template %s: %v", tmplName, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleIndex renders the main page with the list of notes
func handleIndex(w http.ResponseWriter, r *http.Request) {
	notes, err := GetAllNotes()
	if err != nil {
		log.Printf("Error fetching notes: %v", err)
		http.Error(w, "Failed to load notes", http.StatusInternalServerError)
		return
	}

	// Data to pass to the base template
	pageData := map[string]interface{}{
		"Title": "Go Notes App",
		"Notes": notes,
	}
	renderTemplate(w, "base.html", pageData)
}

// handleUpload processes the uploaded markdown file
func handleUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB limit
		http.Error(w, "Could not parse multipart form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("noteFile") // Must match <input name="noteFile">
	if err != nil {
		http.Error(w, "Error retrieving the file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Basic validation
	if !strings.HasSuffix(strings.ToLower(handler.Filename), ".md") {
		http.Error(w, "Invalid file type. Only .md files are allowed.", http.StatusBadRequest)
		return
	}
	if handler.Size == 0 {
		http.Error(w, "Uploaded file is empty.", http.StatusBadRequest)
		return
	}

	log.Printf("Uploaded File: %+v\n", handler.Filename)
	log.Printf("File Size: %+v\n", handler.Size)
	log.Printf("MIME Header: %+v\n", handler.Header)

	// Read file content
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Error reading the file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	markdownContent := string(fileBytes)

	// --- Processing ---
	// 1. Grammar Check
	grammarIssues, err := CheckGrammar(markdownContent)
	if err != nil {
		log.Printf("Grammar check failed for %s: %v", handler.Filename, err)
		// Decide how to handle: fail upload, or proceed without issues?
		// Let's proceed but log the error, maybe add a warning issue.
		grammarIssues = append(grammarIssues, GrammarIssue{Message: "Grammar check process failed: " + err.Error()})
	}

	// 2. Render Markdown to HTML
	htmlContent := RenderMarkdownToHTML(markdownContent)

	// 3. Create Note struct
	newNote := Note{
		OriginalFilename: filepath.Base(handler.Filename), // Basic sanitization
		MarkdownContent:  markdownContent,
		HTMLContent:      htmlContent,
		GrammarIssues:    grammarIssues,
		// ID and CreatedAt will be set by MongoDB driver or CreateNote func
	}

	// 4. Save to Database
	_, err = CreateNote(newNote)
	if err != nil {
		log.Printf("Error saving note to DB: %v", err)
		http.Error(w, "Failed to save the note.", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully processed and saved note: %s", newNote.OriginalFilename)

	// --- HTMX Response ---
	// Instead of redirecting, return the updated list of notes fragment
	// This will replace the content of the target div specified in hx-target
	notes, err := GetAllNotes() // Fetch the fresh list
	if err != nil {
		log.Printf("Error fetching notes after upload: %v", err)
		// Fallback or error message? For simplicity, render empty list on error
		notes = []Note{}
	}

	w.Header().Set("Content-Type", "text/html")
	// Execute *only* the partial template for the note list
	renderTemplate(w, "_notelist.html", notes)

	// --- Alternative: Full Page Redirect (less HTMX-y) ---
	// http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleGetNoteContent fetches a note and renders its content based on 'type' query param
func handleGetNoteContent(w http.ResponseWriter, r *http.Request) {
	noteID := chi.URLParam(r, "id")
	contentType := r.URL.Query().Get("type") // e.g., "html", "markdown", "details" (default)

	if noteID == "" {
		http.Error(w, "Missing note ID", http.StatusBadRequest)
		return
	}

	note, err := GetNoteByID(noteID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Note not found", http.StatusNotFound)
		} else {
			log.Printf("Error fetching note %s: %v", noteID, err)
			http.Error(w, "Error fetching note", http.StatusInternalServerError)
		}
		return
	}

	switch contentType {
	case "markdown":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(note.MarkdownContent)) // Ignore write error for simplicity here
	case "html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(note.HTMLContent)) // Ignore write error
	case "details", "": // Default to showing details fragment
		fallthrough // Explicit fallthrough
	default:
		// Render the detail partial template
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		renderTemplate(w, "_note_detail.html", note)
	}
}

// handleDeleteNote deletes a note and returns the updated list
func handleDeleteNote(w http.ResponseWriter, r *http.Request) {
	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		http.Error(w, "Missing note ID", http.StatusBadRequest)
		return
	}

	err := DeleteNoteByID(noteID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Note already deleted? Still return the current list.
			log.Printf("Attempted to delete non-existent note %s", noteID)
		} else {
			log.Printf("Error deleting note %s: %v", noteID, err)
			http.Error(w, "Error deleting note", http.StatusInternalServerError)
			return
		}
	} else {
		log.Printf("Deleted note %s successfully", noteID)
	}

	// --- HTMX Response ---
	// Return the updated note list fragment to replace the existing list
	notes, err := GetAllNotes()
	if err != nil {
		log.Printf("Error fetching notes after delete: %v", err)
		// Return empty response on error? Or maybe just 200 OK?
		// For HTMX, often best to return the state even on partial failure.
		notes = []Note{} // Render empty list on error
	}

	w.Header().Set("Content-Type", "text/html")
	renderTemplate(w, "_notelist.html", notes)

	// --- Alternative: Simple 200 OK (Less ideal for list updates) ---
	// w.WriteHeader(http.StatusOK)
	// This would require the frontend HTMX to maybe trigger a separate GET on the list
}

// RenderMarkdownToHTML converts markdown string to HTML string
func RenderMarkdownToHTML(mdContent string) string {
	// Configure markdown parser extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse([]byte(mdContent))

	// Configure HTML renderer options
	htmlFlags := html.CommonFlags | html.HrefTargetBlank // Open external links in new tab
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return string(md.Render(doc, renderer))
}
