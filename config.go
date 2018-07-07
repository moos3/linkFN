package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Mailgun struct {
		Domain       string `json:"domain"`
		DomainAPIKey string `json:"domain-api-key"`
		PublicKey    string `json:"public-key"`
	} `json:"mailgun"`
	InfluxDB struct {
		Enabled   string `json:"enabled"`
		Name      string `json:"name"`
		Username  string `json:"username"`
		Password  string `json:"password"`
		Precision string `json:"precision"`
		Host      string `json:"host"`
		Port      string `json:"port"`
	} `json:"influxdb"`
	Port string `json:"port"`
	Host string `json:"host"`
}

func LoadConfiguration(file string) Config {
	var config Config
	configFile, err := os.Open(file)
	defer configFile.Close()
	if err != nil {
		fmt.Println(err.Error())
	}
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)
	return config
}
