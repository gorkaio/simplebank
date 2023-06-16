-- name: CreateTransfer :one
INSERT INTO transfers (
    from_account_id, to_account_id, amount 
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetTransfer :one
SELECT * FROM transfers
WHERE id = $1 LIMIT 1;

-- name: ListTranfers :many
-- TODO this should use sqlc.narg instead of checking zero values, but narg does not work for some reason :/
SELECT * FROM transfers
WHERE
    (CASE WHEN sqlc.arg(from_account_id) != 0 THEN from_account_id = sqlc.arg(from_account_id) ELSE TRUE END) AND
    (CASE WHEN sqlc.arg(to_account_id) != 0 THEN to_account_id = sqlc.arg(to_account_id) ELSE TRUE END)
ORDER BY id
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');