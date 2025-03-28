CREATE TABLE entries (
    id TEXT PRIMARY KEY,
    ask TEXT,
    anon BOOLEAN NOT NULL,
    name TEXT,
    ip TEXT,
    created TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);