package ga

import (
	"log"
	"regexp"
	"yapperbot-frs/src/wikinteract"
	"yapperbot-frs/src/yapperconfig"

	"cgt.name/pkg/go-mwclient"
)

// gaTopics is a map in the form {"subtopic": "topic"}
var gaTopics map[string]string
var gaTopicsRegex *regexp.Regexp
var gaSubtopicRegex *regexp.Regexp

var gaGuidelinesHeaderPageID string = yapperconfig.Config.GAGuidelinesHeaderPageID

func init() {
	gaTopics = map[string]string{
		"Miscellaneous": "Miscellaneous",
	}

	// This regex is used in FetchGATopics to get a list of the subtopics for each topic category
	gaTopicsRegex = regexp.MustCompile(`'''(.*?)'''\s*?<br>\s*?\n((?:\[\[[^|]*\|(?:[^|]*)]](?:{{·}})?\n?)+)`)

	// This regex is used in FetchGATopics to get each subtopic without any trash around it
	gaSubtopicRegex = regexp.MustCompile(`\[\[[^|]*\|([^|]*)]]`)
}

// FetchGATopics takes a mwclient and fetches the latest GA topics from the Good Article noms page.
func FetchGATopics(w *mwclient.Client) {
	text, err := wikinteract.FetchWikitext(w, gaGuidelinesHeaderPageID)
	if err != nil {
		log.Fatal("Failed to fetch Good Articles topics with error ", err)
	}
	matches := gaTopicsRegex.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		// match is in the form [full match, topic, all subtopic links]
		subtopics := gaSubtopicRegex.FindAllStringSubmatch(match[2], -1)
		for _, subtopic := range subtopics {
			// subtopic is in the form [full match, subtopic matched]
			gaTopics[subtopic[1]] = match[1]
		}
	}
}
