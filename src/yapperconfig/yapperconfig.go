package yapperconfig

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
	"encoding/binary"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

type configObject struct {
	FRSPageID                string
	SentCountPageID          string
	GAGuidelinesHeaderPageID string
	APIEndpoint              string
	BotUsername              string
	EditLimit                int64
}

// Config is the global configuration object. This should only really ever be read from.
var Config configObject

var currentUsedEditLimit int64

func init() {
	configFile, err := ioutil.ReadFile("config.yml")
	if err != nil {
		log.Fatal("Config file does not exist, create it!")
	}
	yaml.Unmarshal(configFile, &Config)

	if Config.EditLimit > 0 {
		editLimitFileContents, err := ioutil.ReadFile("editlimit")
		if err != nil {
			// the runfile doesn't exist probably, try creating it
			err := ioutil.WriteFile("editlimit", []uint8{0x00, 0x00, 0x00}, 0644)
			if err != nil {
				log.Fatal("Failed to create edit limit file with error ", err)
			}
			editLimitFileContents = []uint8{0x00, 0x00, 0x00}
		}
		var bytesRead int
		currentUsedEditLimit, bytesRead = binary.Varint(editLimitFileContents)
		if bytesRead < 0 {
			log.Fatal("editlimit file is corrupt, failed to convert with bytesRead ", bytesRead)
		}
	}
}

// EditLimit can be called to increment the current edit count
// Returns true if allowed to edit or false if not
func EditLimit() bool {
	if Config.EditLimit > 0 {
		if currentUsedEditLimit >= Config.EditLimit {
			log.Println("edit limited, not performing edit - limit was", Config.EditLimit, "and this is", currentUsedEditLimit)
			return false
		}

		currentUsedEditLimit++
		return true
	}
	return true
}

// SaveEditLimit saves the current edit limit to the edit limit file
func SaveEditLimit() {
	if Config.EditLimit > 0 {
		buf := make([]byte, binary.MaxVarintLen16)
		binary.PutVarint(buf, currentUsedEditLimit)
		err := ioutil.WriteFile("editlimit", buf, 0644)
		if err != nil {
			log.Fatal("Failed to write edit limit file with err ", err)
		}
	}
}
