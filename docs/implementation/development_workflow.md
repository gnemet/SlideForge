# SlideForge Development Workflow

SlideForge follows specific development patterns to ensure rapid iteration and integration with the wider GoBI/Antigravity library ecosystem.

## Workspace Environment

- **Standard Location**: The project is finalized in `/home/gnemet/GitHub/SlideForge`.
- **Environment Parity**: The `.env` file is critical for local development, defining database connections (`DB_URL`) and AI provider keys (`GEMINI_KEY`, `OPENAI_API_KEY`).

## Local Library Integration

SlideForge leverages the `datagrid` library as a local replacement to ensure synchronization with the latest UI standards.

- **Implementation**: The `go.mod` file uses a `replace` directive.
- **Directive**: `replace github.com/gnemet/datagrid => /home/gnemet/GitHub/datagrid`
- **Benefit**: This allows the "Library-First" architecture to flourish, as enhancements in `datagrid` are immediately visible in SlideForge.

## Asset Workflow

### UI/Styles
- **Branding Assets**: Professional blacksmithing logo and favicon are stored in `ui/static/images/`.
- **Global Styles**: Core design is defined in `ui/static/css/style.css` using the Glassmorphism system inspired by Jiramntr.
- **External Resources**: Fonts and icons are loaded via CDNs (Google Fonts, Phosphor Icons) for development speed.

### Database Migrations
- Schema changes are managed via SQL files in `database/migrations/`.
- PostgreSQL 18 features (like GIN indexes on JSONB) are utilized for performance.

## Build and Execution
- **Script**: `build_run.sh` automates directory creation, build, and execution.
- **Binary**: The application compiles to `bin/slideforge`.
- **Environment**: Managed via `.env` (loaded by `godotenv`).

## Feature Documentation
- [Duplicate Prevention (Checksum)](../features/checksum_logic.md)
- [Generator UI/UX](../features/generator_ui.md)
- [Datagrid Integration](../features/datagrid_integration.md)
