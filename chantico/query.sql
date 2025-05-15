-- name: ListMeasurements :many
SELECT * FROM measurements
ORDER BY name;

-- name: CreateMeasurement :one
INSERT INTO measurements (
	id, name, is_internal, protocol, data_source, query
) VALUES (
	$1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetMeasurement :one
SELECT * FROM measurements WHERE id = $1;

-- name: DeleteMeasurement :exec
DELETE FROM measurements WHERE id = $1;
