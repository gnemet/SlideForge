# SlideForge Technical Stack

SlideForge follows a minimalist, high-performance architecture similar to other GoBI/Antigravity projects.

## Backend
- **Language**: GoLang
- **Frameworks**: None (Vanilla Go)
- **Persistence**: PostgreSQL 18
- **Libraries**:
  - **github.com/russross/blackfriday/v2**: Used for high-performance server-side Markdown-to-HTML rendering.
- **Conversion Utilities**: 
  - **LibreOffice 24.2+**: Used for PPTX to PDF and Template processing.
  - **pdftoppm (poppler-utils)**: Used for high-quality PDF to PNG extraction.
- **Pattern Reference**: `/home/gnemet/GitHub/datagrid`

## Frontend
- **Technology**: HTMX (for dynamic partial updates)
- **Styling**: Standard CSS
- **Scripting**: Vanilla JavaScript and jQuery
- **Pattern**: Hypermedia-driven application design.

## Infrastructure
- **Git Repository**: GitHub (`gnemet/SlideForge`)
- **Development Path**: `/home/gnemet/GitHub/SlideForge`
