package main

import (
	prometheus "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/chantico/prometheus"
	sqlhelper "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/chantico/sql-helper"
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
	StartTimeArg     *string
	DurationArg      *string
	PostgresURL      *string
	MeasurementIDArg *string

	StartTime     time.Time
	MeasurementID pgtype.UUID
	Duration      time.Duration
}

func (args *Args) Process() {
	var err error

	args.StartTimeArg = flag.String("time", "2006-01-02 15:04:05", "the start range of the query")
	args.DurationArg = flag.String("duration", "59m", "the duration")
	args.PostgresURL = flag.String("pgdbstring", os.Getenv("PG_DBSTRING"), "the url of the postgres database - the default value is the PG_DBSTRING env variable")
	args.MeasurementIDArg = flag.String("uuid", "", "the measurement UUID")

	flag.Parse()

	args.StartTime, err = time.Parse("2006-01-02 15:04:05", *args.StartTimeArg)
	if err != nil {
		panic(err)
	}

	args.MeasurementID = *new(pgtype.UUID)
	err = args.MeasurementID.Scan(*args.MeasurementIDArg)
	if err != nil {
		panic(err)
	}

	args.Duration, err = time.ParseDuration(string(*args.DurationArg))
	if err != nil {
		panic(err)
	}

	if args.StartTime.Add(args.Duration).Unix() > time.Now().Unix() {
		log.Println("This end time range is in the future.")
		os.Exit(0)
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
		fmt.Println("Could not create measurement", err)
		return
	}

	// Get the prometheus data
	query_url := fmt.Sprintf(
		"http://%s/api/v1/query_range?query=%s&step=1&start=%d.000&end=%d.000",
		measurement.DataSource,
		url.PathEscape(measurement.Query),
		args.StartTime.Unix(),
		args.StartTime.Add(args.Duration).Unix(),
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
	err = startTimestamp.Scan(args.StartTime)
	if err != nil {
		panic("Start time wrongly formatted")
	}

	endTimestamp := pgtype.Timestamp{}
	err = endTimestamp.Scan(args.StartTime.Add(args.Duration))
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
	fmt.Print("Created measurement: ", measurementValue.ID)
}
