package main

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
)

const DefaultToshlTokenFilename = "toshl-token.dat"

func GetDefaultToshlToken() string {
	return GetToshlToken(DefaultToshlTokenFilename)
}

func GetToshlToken(filename string) string {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("Could not read from token file [%s]: %s", filename, err)
	}

	cleaned := strings.Trim(string(data), "\n")

	return cleaned
}
