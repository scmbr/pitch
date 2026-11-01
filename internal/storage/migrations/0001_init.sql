CREATE TABLE projects (
    id         INTEGER PRIMARY KEY,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL CHECK (type IN ('pptx', 'free')),
    pptx_path  TEXT,
    created_at TEXT NOT NULL
);

CREATE TABLE project_versions (
    id             INTEGER PRIMARY KEY,
    project_id     INTEGER NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    file_hash      TEXT NOT NULL,
    version_number INTEGER NOT NULL,
    created_at     TEXT NOT NULL,
    UNIQUE (project_id, version_number),
    UNIQUE (project_id, file_hash)
);

CREATE TABLE attempts (
    id           INTEGER PRIMARY KEY,
    project_id   INTEGER NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    version_id   INTEGER REFERENCES project_versions (id) ON DELETE SET NULL,
    video_path   TEXT NOT NULL,
    thumb_path   TEXT,
    duration_sec REAL NOT NULL CHECK (duration_sec >= 0),
    topic        TEXT,
    scores       TEXT,
    note         TEXT,
    created_at   TEXT NOT NULL
);

CREATE INDEX attempts_project_created ON attempts (project_id, created_at);

CREATE TABLE slide_timings (
    attempt_id  INTEGER NOT NULL REFERENCES attempts (id) ON DELETE CASCADE,
    slide_index INTEGER NOT NULL CHECK (slide_index >= 0),
    start_sec   REAL NOT NULL CHECK (start_sec >= 0),
    end_sec     REAL NOT NULL,
    PRIMARY KEY (attempt_id, slide_index, start_sec),
    CHECK (end_sec >= start_sec)
);
