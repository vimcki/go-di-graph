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
	Title string `json:"title"`
	Graph Graph  `json:"graph"`
}

type Graph struct {
	Data string `json:"data"`
}

func main() {
	name := "heimdall"
	data, err := os.ReadFile("projects/" + name + "/enhanced.json")
	if err != nil {
		panic(err)
	}

	report := report{
		Report: Report{
			Title: name,
			Graph: Graph{
				Data: string(data),
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
