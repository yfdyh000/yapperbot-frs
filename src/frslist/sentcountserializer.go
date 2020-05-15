package frslist

import (
	"encoding/json"
	"log"

	"github.com/antonholmquist/jason"
)

func serializeSentCount(sc map[string]map[string]int16) string {
	serializedSentCount, err := json.Marshal(sc)
	if err != nil {
		log.Fatal("Failed to serialize sent count, dumping what I was trying to serialize: ", sc)
	}
	return string(serializedSentCount)
}

func deserializeSentCount(json *jason.Object) (sc map[string]map[string]int16) {
	sc = map[string]map[string]int16{} // initialise the map
	headers, err := json.GetObject("headers")
	if err != nil {
		log.Fatal("Failed to deserialize sent count headers, is the JSON invalid?")
	}
	for header, users := range headers.Map() {
		sc[header] = map[string]int16{} // initialise the submap

		users, err := users.Object()
		if err != nil {
			log.Fatal("users wasn't an object, I can't handle this! the JSON seems invalid.")
		}
		for user, count := range users.Map() {
			count, err := count.Int64()
			if err != nil {
				log.Fatal("count wasn't a valid number, I can't handle this! the JSON seems invalid.")
			}
			// this never needs to be an int64, it's just that the library doesn't have arbitrary size int handling
			// converting it back down to int16 at least saves a little memory in the long run, not that it hugely matters
			sc[header][user] = int16(count)
		}
	}
	return
}
