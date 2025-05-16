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
	value REAL NOT NULL,
	timestamp_start TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
	timestamp_end TIMESTAMP NOT NULL
);


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE measurement_values;
DROP TABLE measurements;
-- +goose StatementEnd
