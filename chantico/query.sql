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

-- name: UpdateLastMeasurementTime :one
UPDATE measurements
SET last_measurement_time = $2 
WHERE id = $1
RETURNING *;

-- name: DeleteMeasurement :exec
DELETE FROM measurements WHERE id = $1;

-- name: CreateMeasurementValue :one
INSERT INTO measurement_values (
	measurement_id, value, timestamp_start, timestamp_end
) VALUES (
	$1, $2, $3, $4
)
RETURNING *;

-- name: UpdatePhysicalMeasurement :one
INSERT INTO physical_measurements (
    id, service_id
) VALUES (
    $1, $2
)
ON CONFLICT (service_id) 
DO UPDATE SET
    id = EXCLUDED.id
RETURNING *;
