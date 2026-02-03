# SlideForge Workflow

SlideForge is an AI-powered PPTX orchestrator designed to streamline the analysis and reorganization of presentation content.

## 1. Upload & Extraction
- **Drag & Drop**: Users upload PPTX files through the premium dashboard.
- **Conversion**: The backend uses `LibreOffice` and `pdftoppm` to convert slides into high-resolution PNGs.
- **Persistence**: Metadata and slide paths are stored in PostgreSQL 18.

## 2. Selection & Analysis
- **Grid View**: A glassmorphism-style selection UI displays extracted slides.
- **AI Analysis**: Individual or multiple slides can be analyzed using **Gemini**, **OpenAI**, or **Claude**.
- **Insights**: The AI describes topics, extracted text, and visual styles to help in categorization.

## 3. Smart Stitching (Upcoming)
- **Collection**: Selected slides across different presentations can be added to a "Collection".
- **Generation**: A new PPTX is dynamically "stitched" together based on the collected slides and AI-enhanced structure.

## 4. Multi-Provider AI Support
SlideForge supports dynamic driver switching:
- **Google Gemini**: Default for multimodal analysis.
- **OpenAI**: High-precision text extraction.
- **Anthropic Claude**: Superior reasoning for structural reorganization.
