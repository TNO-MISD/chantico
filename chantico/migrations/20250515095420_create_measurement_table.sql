-- +goose Up
-- +goose StatementBegin
CREATE TABLE measurements (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name VARCHAR(255) NOT NULL,
	is_internal BOOLEAN NOT NULL,
	protocol VARCHAR(255) NOT NULL,
	data_source VARCHAR(255) NOT NULL,
	query TEXT NOT NULL
);

CREATE TABLE measurement_values (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	measurement_id UUID NOT NULL REFERENCES measurements(id),
	value TEXT NOT NULL,
	timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE measurement_values;
DROP TABLE measurements;
-- +goose StatementEnd
