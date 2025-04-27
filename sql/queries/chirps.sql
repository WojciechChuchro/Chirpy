-- name: CreateChirp :one
INSERT INTO chirps (body, user_id)
    VALUES ($1, $2)
RETURNING
    *;

-- name: DeleteAllChirps :exec
DELETE FROM chirps;

-- name: GetAllChrips :many
SELECT
    *
FROM
    chirps
ORDER BY
    created_at;

