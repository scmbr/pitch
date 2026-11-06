ALTER TABLE attempts ADD COLUMN deleting INTEGER NOT NULL DEFAULT 0 CHECK (deleting IN (0,1));
CREATE UNIQUE INDEX attempts_video_path ON attempts(video_path);
