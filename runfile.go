package main

//
// Yapperbot-FRS, the Feedback Request Service bot for Wikipedia
// Copyright (C) 2020 Naypta

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
//

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
