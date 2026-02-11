# SlideForge

AI-Powered PPTX Orchestrator for dynamic slide generation and template management.

## Features
1. **PPTX to Template**: Convert existing presentations into dynamic templates.
2. **Slide Viewer**: Instant PNG generation of all slides for browser viewing.
3. **AI Orchestration**: Slide analysis and metadata extraction using AI.
4. **PostgreSQL 18 Integration**: Robust indexing and search for your PPTX collection.
5. **Modern UI**: Built with Go, HTMX, jQuery, and Phosphor Icons.

## 🚀 Deployment & Distributions

SlideForge is distributed as a **Single-Binary** with all UI assets (templates, CSS, JS) embedded. We provide four pre-configured versions in our [GitHub Releases](https://github.com/gnemet/SlideForge/releases):

1.  **win11online**: Windows 11 version using Google Gemini AI.
2.  **win11offline**: Windows 11 version using **Local LLM (Ollama)** for air-gapped security.
3.  **linuxOn**: Linux x64 version using Cloud AI.
4.  **linuxOff**: Linux x64 version using Local AI.

### 🤖 Local AI (Offline Mode)
For fully private, offline orchestration:
1.  Install [Ollama](https://ollama.com).
2.  Run `ollama pull llama3`.
3.  Use the **offline** distribution; it is pre-configured to point to `localhost:11434`.

---

## 🛠️ Technology Stack
- **Engine**: Pure Go Backend (1.25+)
- **Frontend**: HTMX & Embedded Vanilla CSS (Glassmorphism)
- **Database**: PostgreSQL 17+ with `pgvector`
- **Processing**: LibreOffice & Poppler (pdftoppm)
- **Digital Blacksmith Architecture**: Centralized `config.yaml` and embedded asset orchestration.

## 🏁 Getting Started
1. **Download** the relevant distribution from the [Releases](https://github.com/gnemet/SlideForge/releases) page.
2. **Configure**: Update `config.yaml` with your DB credentials.
3. **Database**: Run the migrations located in `internal/database/migrations/`.
4. **Launch**:
   - **Windows**: Run `RUN.bat`
   - **Linux**: Run `./run.sh`

## Repository Structure
- `cmd/server/`: Main application entry point.
- `internal/`: Core logic and assets.
  - `assets/ui/`: Frontend templates and static files (embedded).
  - `pptx/`: PPTX manipulation logic.
  - `database/`: SQL migrations and repository.
- `bin/`: Compiled binaries.
- `opt/envs/`: Environment configuration management.
- `scripts/`: Maintenance and utility scripts.
- **Dynamic Links** (Created by `build_run.sh`):
  - `uploads/` -> External staging area.
  - `templates/` -> External template storage.
  - `thumbnails/` -> External thumbnail storage.
