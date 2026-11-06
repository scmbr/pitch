ALTER TABLE project_versions ADD COLUMN slide_count INTEGER NOT NULL DEFAULT 0 CHECK(slide_count >= 0);
