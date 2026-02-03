-- Migration to add Title to pptx_files
-- PostgreSQL 18
SET search_path TO slideforge,
    public;
ALTER TABLE pptx_files
ADD COLUMN IF NOT EXISTS title TEXT DEFAULT '';