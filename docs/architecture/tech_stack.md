# 🛠️ Technical Stack: The Foundation of the Foundry

SlideForge is built on a minimalist, high-performance architecture optimized for extreme concurrency and hypermedia-driven interactivity.

## ⚙️ Backend Engine
Our backend is a "Pure Go" implementation, avoiding heavy frameworks for maximum execution speed and maintainability.
-   **Language**: Go (Golang) 1.22+
-   **Data Storage**: **PostgreSQL 17** with `pgvector` for thematic similarity indexing.
-   **Core Libraries**:
    -   `github.com/google/generative-ai-go`: The spinal cord of our AI integration.
    -   `github.com/russross/blackfriday/v2`: High-speed server-side Markdown rendering.
-   **Forensic Capture Utilities**:
    -   **LibreOffice 24.2+ (Headless)**: Precision-grade PPTX decomposition.
    -   **Poppler (pdftoppm)**: High-fidelity (300dpi) PNG extraction from slide atoms.

## 🎨 Hypermedia Frontend
We reject "Single Page Application" (SPA) complexity in favor of high-performance Hypermedia.
-   **Engine**: **HTMX** (Partial DOM swaps for sub-100ms UI responsiveness).
-   **Styling**: **Premium Vanilla CSS** - No heavy CSS frameworks; just pure tokens, glassmorphism, and hardware-accelerated animations.
-   **Icons**: **Phosphor Icons / Font Awesome 6** (Pro-grade visual language).
-   **Interactivity**: **Mermaid JS** for architectural visualization and real-time relationship mapping.

## 🌐 Ecosystem & Infrastructure
-   **AI Integration**: **Digital Blacksmith Engine** (Gemini Multi-Turn Chat Sessions). Implements a **Persona-based Instruction system** (Presentation Architect) for architectural consistency and high-fidelity extraction.
-   **Monitoring**: **Google Cloud Billing API** for real-time financial transparency.
-   **Configuration**: **Zero-Hardcode YAML** (`config.yaml`) with dynamic environment variable fallback.
-   **Storage**: **Cloud-Sync Ingestion** via rclone/inotify for atomic PPTX staging.

---
*2026 | SlideForge Technical Specification*
*Part of the Antigravity Intelligence Ecosystem*
