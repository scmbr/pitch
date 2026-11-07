CREATE TABLE topic_sets (
    id INTEGER PRIMARY KEY,
    set_key TEXT UNIQUE,
    name TEXT NOT NULL,
    language TEXT NOT NULL DEFAULT '',
    builtin INTEGER NOT NULL DEFAULT 0 CHECK(builtin IN (0,1))
);
CREATE TABLE topics (
    id INTEGER PRIMARY KEY,
    set_id INTEGER NOT NULL REFERENCES topic_sets(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    UNIQUE(set_id,body)
);
CREATE TABLE recent_topics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    topic_id INTEGER NOT NULL REFERENCES topics(id) ON DELETE CASCADE
);
