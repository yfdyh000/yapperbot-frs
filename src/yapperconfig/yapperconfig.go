package yapperconfig

import (
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
}

// Config is the global configuration object. This should only really ever be read from.
var Config configObject

func init() {
	configFile, err := ioutil.ReadFile("config.yml")
	if err != nil {
		log.Fatal("Config file does not exist, create it!")
	}
	yaml.Unmarshal(configFile, &Config)
}
