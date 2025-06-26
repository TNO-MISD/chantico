-- +goose Up
-- +goose StatementBegin
CREATE TABLE measurements (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name VARCHAR(255) NOT NULL,
	is_internal BOOLEAN NOT NULL,
	protocol VARCHAR(255) NOT NULL,
	data_source VARCHAR(255) NOT NULL,
	query TEXT NOT NULL,
	registration_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
	last_measurement_time TIMESTAMP
);

CREATE TABLE measurement_values (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	measurement_id UUID NOT NULL REFERENCES measurements(id),
	value REAL NOT NULL,
	timestamp_start TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
	timestamp_end TIMESTAMP NOT NULL
);


CREATE TABLE physical_measurements (
	id UUID NOT NULL,
	service_id VARCHAR(255) PRIMARY KEY
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE measurement_values;
DROP TABLE measurements;
DROP TABLE physical_measurements
-- +goose StatementEnd
