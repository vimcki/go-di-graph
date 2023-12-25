package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type report struct {
	Report Report `json:"report"`
}

type Report struct {
	Title  string        `json:"title"`
	Graph  GraphOrConfig `json:"graph"`
	Config GraphOrConfig `json:"config"`
}

type GraphOrConfig struct {
	Data string `json:"data"`
}

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

	report := report{
		Report: Report{
			Title: name,
			Graph: GraphOrConfig{
				Data: string(graphData),
			},
			Config: GraphOrConfig{
				Data: string(configData),
			},
		},
	}

	serializd, err := json.Marshal(report)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(serializd))

	req, err := http.NewRequest(
		"POST",
		"http://localhost:4000/api/reports",
		bytes.NewReader(serializd),
	)

	req.Header.Set("Content-Type", "application/json")

	if err != nil {
		panic(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.Status)
}
