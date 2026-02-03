# SlideForge Full Functional Test Plan

This document outlines the step-by-step procedures to verify the core functionality of the SlideForge application, covering the end-to-end workflow from file ingestion to slide generation.

## Prerequisities

1.  **Environment**: Properly configured `env` file (e.g., `.env_zenbook`).
2.  **Mounts**: Google Drive or local storage paths accessible and writable.
3.  **Database**: PostgreSQL running with `slideforge_db` schema initialized.
4.  **Application**: Server running (`./build_run.sh zenbook`).

---

## Test Scenarios

### 1. Application Startup & Configuration
**Objective**: Ensure the application starts correctly with the specified environment.

1.  **Stop existing server**: `fuser -k 8088/tcp`
2.  **Run with environment**: `./build_run.sh zenbook`
3.  **Verify Output**:
    *   "Using environment configuration: opt/envs/.env_zenbook"
    *   "Google Drive already mounted..." (or mounting message)
    *   "Database connection established"
    *   "Background observer started, watching: .../stage"
    *   "Starting SlideForge on http://localhost:8088"
4.  **Access UI**: Open `http://localhost:8088` in a browser.
    *   **Pass**: Login page or Dashboard loads successfully.

### 2. File Ingestion (Background Observer)
**Objective**: Verify the "Stage" to "Template" workflow.

1.  **Prepare File**: Have a valid `.pptx` file ready (e.g., `test_presentation.pptx`).
2.  **Copy to Stage**: Copy the file to the configured stage directory.
    *   *Action*: `cp test_presentation.pptx mnt/bdo/stage/`
3.  **Monitor Logs**: Watch the terminal output of the running server.
    *   **Expect**: "Detected change in: ...test_presentation.pptx"
    *   **Expect**: "Processing file: test_presentation.pptx"
    *   **Expect**: "Successfully processed: ..."
    *   **Expect**: "Moved test_presentation.pptx to .../template/test_presentation.pptx"
4.  **Verify Database**:
    *   *Query*: `SELECT filename, original_file_path FROM pptx_files WHERE filename = 'test_presentation.pptx';`
    *   **Pass**: Record exists, and `original_file_path` points to the `template` directory.
5.  **Verify Thumbnails**:
    *   Check `mnt/bdo/thumbnails/test_presentation/`
    *   **Pass**: Directory exists and contains `slide-0001.png`, `slide-0002.png`, etc.

### 3. Dashboard & Search
**Objective**: Verify uploaded files appear and are searchable.

1.  **Navigate to Dashboard**: `http://localhost:8088/dashboard`
2.  **Check Recent Files**:
    *   **Pass**: `test_presentation.pptx` appears in the "Recent Uploads" list.
3.  **Test Search**:
    *   *Action*: Enter a keyword from the slide content into the search bar.
    *   *Action*: Select "Full Text Search" mode.
    *   *Action*: Click Search.
    *   **Pass**: The specific slide containing the text is returned in the results grid.

### 4. Slide Generator UI
**Objective**: Verify the "Remix" workflow.

1.  **Navigate to Generator**: Click "Slide Generator" in the sidebar (`/generator`).
2.  **Verify Source Tree**:
    *   **Pass**: `test_presentation` appears as a collapsible node in the left pane.
3.  **Expand Node**: Click the chevron or name.
    *   **Pass**: Slides appear listed as "Slide 1", "Slide 2", etc. (or custom titles if AI extracted them).
    *   **Pass check**: The presentation filename prefix is **NOT** present in the individual slide items (per Step 460).
4.  **Preview Slide**: Alt-Click a slide item.
    *   **Pass**: A modal opens showing the high-res image of the slide.
5.  **Metadata**: Shift-Click a slide item.
    *   **Pass**: A modal opens showing AI Summary and Raw Content JSON.

### 5. Collection & Generation
**Objective**: Build a new deck from selected slides.

1.  **Drag & Drop**:
    *   *Action*: Drag "Slide 1" from the left pane to the "Your Collection" (right) pane.
    *   **Pass**: Slide appears in the collection list.
2.  **Multi-Select**:
    *   *Action*: Ctrl-Click multiple slides in the left pane.
    *   *Action*: Drag the group to the right pane.
    *   **Pass**: All selected slides are added to the collection.
3.  **Reorder**:
    *   *Action*: Drag slides up/down within the "Your Collection" pane.
    *   **Pass**: Order updates visually.
4.  **Generate**:
    *   *Action*: Click the "Generate" button (Magic Wand).
    *   **Pass**: Button shows "Generating..." spinner.
    *   **Pass**: (Currently simulated) Alert confirms generation logic initiated.

### 6. Admin & System
**Objective**: Verify system health and settings.

1.  **Config Check**:
    *   Check `config.yaml` or `/admin/config` (if implemented).
    *   **Pass**: Storage paths match the active environment (`mnt/bdo/stage`, etc.).

---

## Troubleshooting

*   **Observer not picking up files**: Ensure `inotify` limits are sufficient on Linux (`fs.inotify.max_user_watches`).
*   **Permissions**: Ensure the user running `./build_run.sh` has write access to `mnt/bdo` (Google Drive mount).
*   **Database Errors**: Check `opt/envs/.env_zenbook` DB credentials and ensure the `slideforge_db` exists.
