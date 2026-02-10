-- Migration to add comments to slides and include them in FTS
SET search_path TO slideforge,
    public;
-- 1. Add comments column to collected_slides if not exists
DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'slideforge'
        AND table_name = 'collected_slides'
        AND column_name = 'comments'
) THEN
ALTER TABLE collected_slides
ADD COLUMN comments TEXT DEFAULT '';
END IF;
END $$;
-- 2. Update the trigger function to include comments in tsvectors
CREATE OR REPLACE FUNCTION update_slide_search_vectors() RETURNS trigger AS $$ BEGIN NEW.fts_en := to_tsvector(
        'english',
        coalesce(NEW.content, '') || ' ' || coalesce(NEW.comments, '')
    );
NEW.fts_hu := to_tsvector(
    'hungarian',
    unaccent(
        coalesce(NEW.content, '') || ' ' || coalesce(NEW.comments, '')
    )
);
NEW.fts_combined := setweight(
    to_tsvector(
        'english',
        coalesce(NEW.content, '') || ' ' || coalesce(NEW.comments, '')
    ),
    'A'
) || setweight(
    to_tsvector(
        'hungarian',
        unaccent(
            coalesce(NEW.content, '') || ' ' || coalesce(NEW.comments, '')
        )
    ),
    'A'
);
RETURN NEW;
END $$ LANGUAGE plpgsql;
-- 3. Re-generate FTS vectors for existing slides (optional but good for consistency)
UPDATE collected_slides
SET comments = COALESCE(comments, '');