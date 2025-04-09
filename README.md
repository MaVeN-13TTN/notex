# Notex - Markdown Note-Taking Application

Notex is a simple, modern note-taking application built with Go and HTMX that supports Markdown and includes grammar checking capabilities.

## Features

- **Markdown Support**: Write notes in Markdown and view rendered HTML
- **Grammar Checking**: Integrated with LanguageTool for grammar and spelling corrections
- **Real-time UI Updates**: Using HTMX for a dynamic experience without complex JavaScript
- **MongoDB Database**: Reliable storage and retrieval of notes

## Tech Stack

- **Backend**: Go with Chi router
- **Frontend**: HTML templates with HTMX
- **Database**: MongoDB
- **Markdown**: gomarkdown/markdown library
- **Grammar Checking**: LanguageTool (Java-based external dependency)

## Prerequisites

- Go 1.20+
- MongoDB
- Java Runtime Environment (for LanguageTool)
- LanguageTool JAR file

## Setup

1. Clone the repository:

   ```
   git clone https://github.com/yourusername/notex.git
   cd notex
   ```

2. Install Go dependencies:

   ```
   go mod download
   ```

3. Set up LanguageTool for grammar checking:

   ```
   # Create external directory
   mkdir -p external

   # Download LanguageTool
   wget https://languagetool.org/download/LanguageTool-stable.zip -O external/LanguageTool-stable.zip

   # Unzip LanguageTool
   unzip external/LanguageTool-stable.zip -d external/
   ```

4. Configure the application:
   Copy the `.env.example` file to `.env` and update the settings:

   ```
   cp .env.example .env
   # Edit .env with your settings, especially LANGUAGETOOL_JAR_PATH
   ```

5. Run the application:

   ```
   go run main.go
   ```

6. Access the application:
   Open your browser and navigate to `http://localhost:8080`

## Project Structure

```
notex/
├── go.mod            # Go module definition
├── go.sum            # Go module checksums
├── main.go           # Main application setup and server start
├── handlers.go       # HTTP handler functions
├── db.go             # MongoDB interaction logic
├── models.go         # Data structure definitions
├── grammar.go        # Grammar checking logic
├── templates/        # HTML templates
│   ├── base.html     # Base layout
│   ├── index.html    # Main page content
│   ├── _notelist.html    # Partial for rendering the list of notes
│   └── _note_detail.html # Partial for rendering note details/content
└── static/           # Static files
    └── htmx.min.js   # HTMX library
```

## Usage

1. **Create a Note**: Click the "New Note" button and start typing in Markdown
2. **Edit a Note**: Click on any note from the list to edit its content
3. **Check Grammar**: Use the "Check Grammar" button to verify spelling and grammar
4. **Format Text**: Use Markdown syntax for formatting (e.g., # for headings, \*\* for bold)

## License

MIT License
