# SlideForge

AI-Powered PPTX Orchestrator for dynamic slide generation and template management.

## Features
1. **PPTX to Template**: Convert existing presentations into dynamic templates.
2. **Slide Viewer**: Instant PNG generation of all slides for browser viewing.
3. **AI Orchestration**: Slide analysis and metadata extraction using AI.
4. **PostgreSQL 18 Integration**: Robust indexing and search for your PPTX collection.
5. **Modern UI**: Built with Go, HTMX, jQuery, and Phosphor Icons.

## Technology Stack
- **Backend**: Go (GoLang)
- **Frontend**: HTMX, Vanilla CSS, JS, jQuery
- **Database**: PostgreSQL 18
- **Processing**: LibreOffice, Poppler (pdftoppm)

## Prerequisites
- Go 1.25+
- LibreOffice (for headless conversion)
- Poppler-utils (for `pdftoppm`)
- PostgreSQL 18

## Getting Started
1. Clone the repository.
2. Setup your `.env` file with your database URL and AI keys.
3. Run the migrations:
   ```bash
   psql slideforge < database/migrations/001_init.sql
   ```
4. Build and run:
   ```bash
   ./build_run.sh
   ```

## Repository Structure
- `cmd/server/`: Main application.
- `internal/`: Core logic (pptx manipulation, ai, db).
- `ui/`: Frontend assets and templates.
- `database/`: SQL migrations and schemas.
- `data/`: Storage for uploads and generated thumbnails.
