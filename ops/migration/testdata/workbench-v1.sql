CREATE TABLE project_profiles (
    repo_id INTEGER PRIMARY KEY,
    stage TEXT NOT NULL CHECK (stage IN ('planned', 'development', 'testing', 'released', 'paused')),
    progress INTEGER NOT NULL CHECK (progress BETWEEN 0 AND 100),
    owner_user_id INTEGER,
    start_date TEXT,
    target_date TEXT,
    risk TEXT NOT NULL CHECK (risk IN ('normal', 'attention', 'blocked')),
    summary TEXT NOT NULL DEFAULT '' CHECK (length(summary) <= 500),
    version INTEGER NOT NULL CHECK (version > 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    updated_by INTEGER NOT NULL CHECK (updated_by > 0)
);

CREATE TABLE project_followers (
    repo_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL CHECK (user_id > 0),
    PRIMARY KEY (repo_id, user_id),
    FOREIGN KEY (repo_id) REFERENCES project_profiles(repo_id) ON DELETE CASCADE
);

CREATE TABLE project_audit_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_id INTEGER NOT NULL,
    actor_user_id INTEGER NOT NULL CHECK (actor_user_id > 0),
    request_id TEXT NOT NULL UNIQUE,
    changed_fields TEXT NOT NULL,
    before_value TEXT,
    after_value TEXT NOT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY (repo_id) REFERENCES project_profiles(repo_id) ON DELETE RESTRICT
);

INSERT INTO project_profiles (
    repo_id, stage, progress, owner_user_id, start_date, target_date, risk, summary,
    version, created_at, updated_at, updated_by
) VALUES (
    3, 'development', 45, 4, '2026-07-01', '2026-09-30', 'attention', 'Legacy project summary',
    3, '2026-07-01T08:00:00Z', '2026-08-01T09:30:00Z', 2
);

INSERT INTO project_followers (repo_id, user_id) VALUES (3, 2), (3, 4);

INSERT INTO project_audit_events (
    id, repo_id, actor_user_id, request_id, changed_fields, before_value, after_value, created_at
) VALUES
    (1, 3, 4, 'legacy-request-1', '["progress"]',
     '{"repo_id":3,"progress":20}', '{"repo_id":3,"progress":45}', '2026-07-15T10:00:00Z'),
    (2, 3, 999, 'legacy-request-2', '["risk"]',
     '{"repo_id":3,"risk":"normal"}', '{"repo_id":3,"risk":"attention"}', '2026-08-01T09:30:00Z');
