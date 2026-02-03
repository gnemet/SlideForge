-- Migration to add Title to collected_slides
-- PostgreSQL 18
SET search_path TO slideforge,
    public;
ALTER TABLE collected_slides
ADD COLUMN IF NOT EXISTS title TEXT DEFAULT '';