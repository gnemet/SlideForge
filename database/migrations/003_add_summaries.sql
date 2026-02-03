-- Migration to add AI summaries for PPTX and Slides
-- PostgreSQL 18
SET search_path TO slideforge,
    public;
ALTER TABLE pptx_files
ADD COLUMN IF NOT EXISTS ai_summary TEXT DEFAULT '';
ALTER TABLE collected_slides
ADD COLUMN IF NOT EXISTS ai_summary TEXT DEFAULT '';