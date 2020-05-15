package rfc

import (
	"fmt"
	"log"
	"regexp"
	"yapperbot-frs/src/wikinteract"

	"cgt.name/pkg/go-mwclient"
	"cgt.name/pkg/go-mwclient/params"
)

// MarkRfcsDone takes a mwclient, a pageid and a series of RfC objects,
// and marks the RfCs with the "frsdone" tag.
func MarkRfcsDone(w *mwclient.Client, pageID string, rfcsDone []RfC) {
	// We want to fetch the content again here, to try and prevent edit conflicts as much as possible
	content, err := wikinteract.FetchWikitext(w, pageID)
	if err != nil {
		log.Fatal("Couldn't get the RfC page again to mark as done - error was ", err)
	}

	for _, rfc := range rfcsDone {
		var rfcTagRegex *regexp.Regexp
		rfcTagRegex, err := regexp.Compile(fmt.Sprintf(`(?i){{rfc((?:.*?)\|rfcid=%s(?:.*?))}}`, rfc.ID))
		if err != nil {
			log.Fatal("Failed to compile RfC tag matcher regex for page ID ", pageID, "RfC ID ", rfc.ID, ", error was: ", err)
		}

		content = rfcTagRegex.ReplaceAllString(content, "{{Rfc$1|frsdone=true}}")
	}

	err = w.Edit(params.Values{
		"pageid":  pageID,
		"summary": "FRS processing for page complete, marking RfC(s) as frsdone",
		"minor":   "true",
		"bot":     "true",
		"text":    content,
	})
	if err != nil {
		log.Fatal("Failed to update RfC page ", pageID, " to mark as done, with error ", err)
	}
}
