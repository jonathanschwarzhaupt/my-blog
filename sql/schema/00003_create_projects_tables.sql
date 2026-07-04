-- +goose Up
CREATE TABLE projects (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name text NOT NULL,
    slug text NOT NULL UNIQUE,
    description text NOT NULL DEFAULT ''
);

CREATE TABLE post_projects (
    post_id bigint NOT NULL REFERENCES posts (id) ON DELETE CASCADE,
    project_id bigint NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, project_id)
);

CREATE INDEX post_projects_project_id_idx ON post_projects (project_id);

-- +goose Down
DROP TABLE IF EXISTS post_projects;
DROP TABLE IF EXISTS projects;
