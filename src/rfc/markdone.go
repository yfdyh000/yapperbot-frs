package rfc

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
	"fmt"
	"log"
	"regexp"
	"yapperbot-frs/src/yapperconfig"

	"cgt.name/pkg/go-mwclient"
	"cgt.name/pkg/go-mwclient/params"
	"github.com/mashedkeyboard/ybtools"
)

// MarkRfcsDone takes a mwclient, a pageid and a series of RfC objects,
// and marks the RfCs with the "frsdone" tag.
func MarkRfcsDone(w *mwclient.Client, pageID string, rfcsDone []RfC) {
	// We want to fetch the content again here, to try and prevent edit conflicts as much as possible
	content, err := ybtools.FetchWikitext(w, pageID)
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

	if yapperconfig.EditLimit() {
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
	} else {
		log.Println("WARNING: EDIT LIMITED OUT OF MARKING RFC AS FRSDONE ON PAGE ID", pageID)
	}
}
