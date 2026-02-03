# 🪟 Windows 11 Deployment Guide (On/Offline)

This guide details how to deploy SlideForge on Windows 11, including a fully **Offline Mode** with local AI (Ollama).

## 🏢 Distribution Architecture
SlideForge uses **Embedded Assets**. This means the `.exe` contains all HTML templates, CSS, JS, and translation files. You only need to distribute the binary and the `config.yaml`.

---

## 🛠️ Prerequisites (Windows side)

To run SlideForge on Windows, the following "Digital Blacksmith" tools are required in your `PATH`:

1.  **PostgreSQL 17**:
    *   [Download Installer](https://www.postgresql.org/download/windows/) or use a portable version.
    *   Ensure `pgvector` extension is available (included in standard enterpriseDB installers).
2.  **LibreOffice 24.2+**:
    *   Required for PPTX decomposition.
    *   [Download](https://www.libreoffice.org/download/download/).
    *   Add `C:\Program Files\LibreOffice\program` to your System PATH.
3.  **Poppler (pdftoppm)**:
    *   Required for slide thumbnail generation.
    *   [Download for Windows](https://github.com/oschwartz10612/poppler-windows/releases).
    *   Add the `bin` folder to your System PATH.

---

## 🤖 Local AI (Offline Mode)

For a fully offline experience, use **Ollama**:

1.  **Install Ollama**: Download from [ollama.com](https://ollama.com).
2.  **Pull Models**: Open PowerShell and run:
    ```powershell
    ollama pull llama3
    ollama pull sqlcoder
    ```
3.  **Configure SlideForge**: Update `config.yaml` to point to the local provider:
    ```yaml
    ai:
      active_provider: "local"
      providers:
        local:
          driver: "openai_compatible"
          endpoint: "http://localhost:11434/v1"
          model: "llama3"
    ```

---

## 🚀 Deployment Steps

### 1. Build the Distribution (on Linux/WSL)
Run the build script to generate the Windows bundle:
```bash
./scripts/build_win.sh
```
This creates a `dist_win/` directory.

### 2. File Structure
The bundle should look like this:
```text
dist_win/
├── SlideForge.exe      (The main engine)
├── RUN.bat             (Easy launcher)
├── config.yaml         (Configuration)
├── uploads/            (Folder for PPTX storage)
└── thumbnails/         (Folder for generated slides)
```

### 3. Initialize Database
Create the database and schema on your Windows Postgres instance:
```sql
CREATE DATABASE slideforge;
-- Run migrations from database/migrations/
```

### 4. Hardware Acceleration
SlideForge is optimized for Windows 11. If you have an NVIDIA GPU, Ollama will automatically use CUDA for the offline LLM, providing sub-second inference speeds.

---

## 🔒 Offline Guarantees
- **Privacy**: No slide data leaves the local machine.
- **Independence**: No connection to Google Cloud or OpenAI is required.
- **Persistence**: All data is stored in the local PostgreSQL instance.

---
*2026 | SlideForge Deployment Specifications*
