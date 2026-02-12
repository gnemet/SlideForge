-- migration: 009_placeholder_discovery
CREATE TABLE IF NOT EXISTS placeholder_discovery (
    id SERIAL PRIMARY KEY,
    pptx_file_id INTEGER REFERENCES pptx_files(id) ON DELETE CASCADE,
    slide_number INTEGER NOT NULL,
    placeholder_text TEXT NOT NULL,
    metadata_key TEXT NOT NULL,
    discovered_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pptx_file_id, slide_number, placeholder_text)
);
COMMENT ON TABLE placeholder_discovery IS 'Stores mappings between slide text and metadata keys as determined by the user.';