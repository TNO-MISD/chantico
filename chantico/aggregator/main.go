package main

import (
	prometheus "chantico/chantico/prometheus"
	sqlhelper "chantico/chantico/sql-helper"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"strconv"
	"time"

	"net/http"
	"net/url"

	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	_ "github.com/lib/pq"
)

type Args struct {
	DurationArg      *string
	PostgresURL      *string
	MeasurementIDArg *string

	MeasurementID pgtype.UUID
	Duration      time.Duration
}

func (args *Args) Process() {
	var err error

	args.DurationArg = flag.String("duration", "1h", "the duration")
	args.PostgresURL = flag.String("pgdbstring", os.Getenv("PG_DBSTRING"), "the url of the postgres database - the default value is the PG_DBSTRING env variable")
	args.MeasurementIDArg = flag.String("uuid", "", "the measurement UUID")

	flag.Parse()

	args.MeasurementID = *new(pgtype.UUID)
	err = args.MeasurementID.Scan(*args.MeasurementIDArg)
	if err != nil {
		panic(err)
	}

	args.Duration, err = time.ParseDuration(string(*args.DurationArg))
	if err != nil {
		panic(err)
	}
}

var retentionDuration time.Duration = (5 * time.Hour)

func UpdateMeasurement(measurement sqlhelper.Measurement, args Args, ctx context.Context, db sqlhelper.DBTX) error {
	queries := sqlhelper.New(db)

	// Get measurement by id
	measurement, err := queries.GetMeasurement(ctx, args.MeasurementID)
	if err != nil {
		fmt.Println("Could not create measurement", err)
		return err
	}

	// Add logic about to determine the start and end time
	maxEndTime := time.Now().UTC()
	minStartTime := maxEndTime.Add(-retentionDuration).Round(time.Hour)
	if minStartTime.Before(measurement.RegistrationTime.Time) {
		minStartTime = minStartTime.Add(time.Hour)
	}

	startTime := measurement.RegistrationTime.Time.Round(time.Hour)
	if startTime.Before(measurement.RegistrationTime.Time) {
		startTime = startTime.Add(time.Hour)
	}
	if measurement.LastMeasurementTime.Time.After(startTime) {
		startTime = measurement.LastMeasurementTime.Time
	}
	if startTime.Before(minStartTime) {
		startTime = minStartTime
	}

	for {
		endTime := startTime.Add(args.Duration).Add(-time.Second)
		if endTime.After(maxEndTime) {
			log.Printf("No more measurement, gracefully shutting down")
			os.Exit(0)
		}

		// Get the prometheus data
		query_url := fmt.Sprintf(
			"http://%s/api/v1/query_range?query=%s&step=1&start=%d.000&end=%d.000",
			measurement.DataSource,
			url.PathEscape(measurement.Query),
			startTime.UTC().Unix(),
			endTime.UTC().Unix(),
		)

		resp, err := http.Get(query_url)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		requestResponse := &prometheus.PrometheusRequestResponse{}
		requestResponse.Parse(body)

		var acc int64 = 0
		n := 0
		for _, value := range (requestResponse.Data.Result[0]).Values {
			if len(value) != 2 {
				log.Fatal("Not enough values")
			}
			valueString := value[1]
			valueInt, err := strconv.ParseInt(valueString, 10, 64)
			if err != nil {
				log.Fatal("values wrongly formatted")
			}
			acc += valueInt
			n++
		}

		// Push the data to Postgres
		startTimestamp := pgtype.Timestamp{}
		err = startTimestamp.Scan(startTime)
		if err != nil {
			panic("Start time wrongly formatted")
		}

		endTimestamp := pgtype.Timestamp{}
		err = endTimestamp.Scan(endTime)
		if err != nil {
			panic("Duration wrongly formatted")
		}

		measurementValueParams := sqlhelper.CreateMeasurementValueParams{
			MeasurementID:  args.MeasurementID,
			Value:          float32(acc) / float32(n),
			TimestampStart: startTimestamp,
			TimestampEnd:   endTimestamp,
		}
		measurementValue, err := queries.CreateMeasurementValue(ctx, measurementValueParams)

		// Update last measurement
		updateLastMeasurementParams := sqlhelper.UpdateLastMeasurementTimeParams{
			ID:                  args.MeasurementID,
			LastMeasurementTime: endTimestamp,
		}
		_, err = queries.UpdateLastMeasurementTime(ctx, updateLastMeasurementParams)

		if err != nil {
			panic("Could not update last measurement")
		}

		log.Printf("Created measurement: %s\n", measurementValue.ID)
		startTime = endTime.Add(time.Second)
	}
}

func main() {
	var err error

	// Parse arguments
	args := Args{}
	args.Process()

	// Initialize database connection
	ctx := context.Background()
	db, err := pgx.Connect(ctx, *args.PostgresURL)
	if err != nil {
		log.Fatal(fmt.Sprintf("Could not connect to db: %s\n", err))
	}
	defer db.Close(ctx)
	queries := sqlhelper.New(db)

	// Get measurement by id
	measurement, err := queries.GetMeasurement(ctx, args.MeasurementID)
	if err != nil {
		log.Fatal(fmt.Sprintf("Could not create measurement: %s\n", err))
	}
	err = UpdateMeasurement(measurement, args, ctx, db)
	if err != nil {
		log.Fatal(fmt.Sprintf("Could not create measurement: %s\n", err))
	}
}
