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
	"io/ioutil"
	"log"

	"github.com/mashedkeyboard/ybtools"
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

func init() {
	configFile, err := ioutil.ReadFile("config.yml")
	if err != nil {
		log.Fatal("Config file does not exist, create it!")
	}
	yaml.Unmarshal(configFile, &Config)

	if Config.EditLimit > 0 {
		ybtools.SetupEditLimit(Config.EditLimit)
	}
}
