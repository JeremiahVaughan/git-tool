CREATE TABLE IF NOT EXISTS repo (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	url TEXT UNIQUE,
	trunk_branch TEXT);

CREATE TABLE IF NOT EXISTS effort (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT UNIQUE,
	branch_name TEXT UNIQUE,
	description TEXT);

CREATE TABLE IF NOT EXISTS effort_repo (
	effort_id INTEGER,
	repo_id INTEGER,
	PRIMARY KEY (effort_id, repo_id),
	FOREIGN KEY (effort_id) REFERENCES effort(id) ON DELETE CASCADE,
	FOREIGN KEY (repo_id) REFERENCES repo(id) ON DELETE CASCADE
);

CREATE INDEX idx_effort_id ON effort_repo (effort_id);                  
