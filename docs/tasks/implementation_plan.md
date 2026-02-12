# PPTX Processing and Generation Plan

The goal is to automate the extraction, analysis, and generation of PPTX files based on a "seed" template and metadata.

## Proposed Changes

### [Python Script]
A Python script `pptx_processor.py` will be created to:
- Unzip all PPTX files from `source` to `unpack`.
- Analyze the XML structure of the PPTX files (specifically `ppt/slides/slide*.xml` and `ppt/presentation.xml`).
- Create a `seed.pptx` by taking a representative slide and stripping it of specific content while keeping common text/placeholders.
- Generate new PPTX files by merging the `seed.pptx` with JSON metadata.

### [Metadata]
- Create `metadata.json` for testing, containing key-value pairs to replace placeholders in the seed PPTX.

### [MCP/Knowledge Skill]
- Create a new skill in `.agent/skills/pptx_generation.md` (or similar) that documents the process.

## Verification Plan

### Automated Tests
- Run `pptx_processor.py` and verify that files are created in `unpack`, `seed`, `metadata`, and `output` directories.
- Check the integrity of the generated PPTX by attempting to list its contents (unzip -l).

### Manual Verification
- The user can open the generated PPTX in LibreOffice to verify the visual layout and text replacement.
- `libreoffice --version` to check if it's available.
