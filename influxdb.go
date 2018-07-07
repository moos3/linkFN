package main

import (
	"fmt"
	"log"
	"time"

	client "github.com/influxdata/influxdb/client/v2"
)

// ******* INFLUX DB SETTINGS *****

// enable influx db
var influxOn = config.InfluxDB.Enabled
var influxDBName = config.InfluxDB.Name
var influxDBUsername = config.InfluxDB.Username
var influxDBPassword = config.InfluxDB.Password
var influxDBPrecision = config.InfluxDB.Precision
var influxDBHost = config.InfluxDB.Host

// **** Begin InfluxDB HERE ******

func statHandler(tags map[string]string, fields map[string]interface{}, url string) {
	c := influxDBClient()
	createMetrics(c, tags, fields, url)
}

func influxDBClient() client.Client {
	fmt.Println(config.InfluxDB.Host)
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     config.InfluxDB.Host,
		Username: config.InfluxDB.Username,
		Password: config.InfluxDB.Password,
	})
	if err != nil {
		log.Fatalln("Error: ", err)
	}

	return c
}

func createMetrics(c client.Client, tags map[string]string, fields map[string]interface{}, urlShort string) {
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  config.InfluxDB.Name,
		Precision: config.InfluxDB.Precision,
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
