package main

import (
	"log"
	"os"
	"time"

	client "github.com/influxdata/influxdb/client/v2"
)

// ******* INFLUX DB SETTINGS *****

// enable influx db
var influxOn string = os.Getenv("INFLUX_DB_ENABLE")
var influxDBName string = os.Getenv("INFLUX_DB_NAME")
var influxDBUsername string = os.Getenv("INFLUX_DB_USERNAME")
var influxDBPassword string = os.Getenv("INFLUX_DB_PASSWORD")
var influxDBPrecision string = os.Getenv("INFLUX_DB_PRECISION")
var influxDBHost string = os.Getenv("INFLUX_DB_HOST")

// **** Begin InfluxDB HERE ******

func influxDBClient() client.Client {
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     influxDBHost,
		Username: influxDBUsername,
		Password: influxDBPassword,
	})
	if err != nil {
		log.Fatalln("Error: ", err)
	}

	return c
}

func createMetrics(c client.Client, tags map[string]string, fields map[string]string, urlShort string) {
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  influxDBName,
		Precision: influxDBPrecision,
	})
	if err != nil {
		log.Fatalln("Error: ", err)
	}

	point, err := client.NewPoint(
		urlShort,
		tags,
		fields,
		time.Now(),
	)
	if err != nil {
		log.Fatalln("Error: ", err)
	}
	bp.AddPoint(point)
	err = c.Write(bp)
	if err != nil {
		log.Fatal(err)
	}

}
