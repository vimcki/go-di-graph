package main

import (
	"os"

	"github.com/vimcki/go-di-graph/internal/report"
)

func main() {
	name := "freya"
	graphData, err := os.ReadFile("_projects/" + name + "/enhanced.json")
	if err != nil {
		panic(err)
	}

	configData, err := os.ReadFile("_projects/" + name + "/config.json")
	if err != nil {
		panic(err)
	}

	report.SendReport(
		"freya",
		"http://localhost:4000/api/reports",
		string(graphData),
		string(configData),
	)
}
