# Duplicate File Prevention (Checksum Logic)

## Overview
SlideForge implements a robust duplicate prevention mechanism using SHA256 checksums to ensure file integrity and prevent redundant processing.

## Implementation Details

### Database Schema
A `checksum` column (VARCHAR(64)) has been added to the `pptx_files` table with a UNIQUE constraint index.

```sql
ALTER TABLE pptx_files ADD COLUMN checksum VARCHAR(64);
CREATE UNIQUE INDEX idx_pptx_files_checksum ON pptx_files(checksum);
```

### Processing Workflow
1.  **Upload/Scan**: When a file is uploaded or detected by the Observer.
2.  **Calculation**: The system calculates the SHA256 hash of the file content.
3.  **Verification**:
    *   The system queries the database for an existing record with this checksum.
    *   **If Found**: The file is identified as a duplicate.
        *   The existing record ID is logged.
        *   Processing is skipped (idempotency).
        *   If it's a metadata update (re-upload of same file), the system updates the metadata but clears old slides to prevent duplication before reprocessing.
    *   **If New**: The file is processed normally, and the checksum is stored.

## Benefits
*   **Storage Efficiency**: prevents storing multiple copies of the same presentation.
*   **Integrity**: Ensures that two files with different names but identical content are treated as the same entity.
