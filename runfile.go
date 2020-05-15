package main

import (
	"io/ioutil"
	"log"
	"strings"

	"github.com/metal3d/go-slugify"
)

func loadFromRunfile(category string) (string, string) {
	var startRunfile []byte
	// runfile stores the last categorisation timestamp
	runfileName := slugify.Marshal(category) + ".frsrunfile"
	startRunfile, err := ioutil.ReadFile(runfileName)
	if err != nil {
		// the runfile doesn't exist probably, try creating it
		err := ioutil.WriteFile(runfileName, []byte(""), 0644)
		if err != nil {
			log.Fatal("Failed to create runfile with error ", err)
		}
		return "", ""
	}
	splitStartRunfile := strings.SplitN(string(startRunfile), ";", 2)

	switch len(splitStartRunfile) {
	case 1:
		splitStartRunfile = append(splitStartRunfile, "")
	case 2:
		break
	default:
		log.Fatal("Corrupt runfile for category ", category)
	}

	return splitStartRunfile[0], splitStartRunfile[1]
}
