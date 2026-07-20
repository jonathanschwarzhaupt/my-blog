-- name: ListSkillsByOrder :many
SELECT * FROM skill ORDER BY order_key ASC, id ASC;

-- name: DeleteAllSkills :exec
DELETE FROM skill;

-- name: InsertSkill :one
INSERT INTO skill (category, name, order_key) VALUES ($1, $2, $3) RETURNING *;
