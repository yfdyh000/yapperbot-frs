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
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"yapperbot-frs/src/frslist"
	"yapperbot-frs/src/rfc"

	"cgt.name/pkg/go-mwclient"
	"cgt.name/pkg/go-mwclient/params"
	"github.com/mashedkeyboard/ybtools"
)

const maxMsgsToSend int = 15
const minMsgsToSend int = 5

var commentRegex *regexp.Regexp

// in the below string, %s represents the header the user was subscribed to
const editSummaryForFeedbackMsgs string = `A [[WP:FRS|Feedback Request Service]] notification for you, powered by Yapperbot. You're getting this message because you subscribed to "%s". To unsubscribe, just remove yourself from the list.`

func init() {
	commentRegex = regexp.MustCompile(`\s*?<!--.*?-->\s*?`)
}

func requestFeedbackFor(requester frsRequesting, w *mwclient.Client) {
	var msgsToSend int = (rand.Intn(maxMsgsToSend-minMsgsToSend) + minMsgsToSend) // evaluates out to any number between max and min
	// it's important that this is a separate array, as we later consider its length
	var headersToSendTo []string

	for _, header := range frslist.GetListHeaders() {
		if requester.IncludeHeader(header) {
			headersToSendTo = append(headersToSendTo, header)
		}
	}

	if len(headersToSendTo) > 0 {
		var messagesPerHeader int = msgsToSend / len(headersToSendTo) // auto-rounds down where needed, as they're integers
		headerusers := frslist.GetUsersFromHeaders(headersToSendTo, messagesPerHeader)

		var textBuilder strings.Builder
		textBuilder.WriteString("{{subst:User:Yapperbot/FRS notification|title=")
		textBuilder.WriteString(requester.PageTitle())
		textBuilder.WriteString("|type=")
		textBuilder.WriteString(requester.RequestType())
		if rfc, isRfC := requester.(rfc.RfC); isRfC {
			textBuilder.WriteString("|rfcid=")
			textBuilder.WriteString(rfc.ID)
		}
		textBuilder.WriteString("}} ~~~~")
		var notificationText string = textBuilder.String()

		for header, users := range headerusers {
			cleanedHeader := commentRegex.ReplaceAllString(header, "")
			editsummary := fmt.Sprintf(editSummaryForFeedbackMsgs, cleanedHeader)
			sectiontitle := fmt.Sprintf("Feedback request: %s %s", cleanedHeader, requester.RequestType())

			for _, user := range users {
				// Drop a note on each user's talk page inviting them to participate
				if ybtools.EditLimit() {
					// the redirect param here automatically resolves redirects,
					// for instance if a user changes their username but forgets
					// to update the FRS user tag
					err := w.Edit(params.Values{
						"title":        "User talk:" + user.Username,
						"section":      "new",
						"sectiontitle": sectiontitle,
						"summary":      editsummary,
						"notminor":     "true",
						"bot":          "true",
						"text":         notificationText,
						"redirect":     "true",
					})
					if err == nil {
						log.Println("Successfully invited", user.Username, "to give feedback on page", requester.PageTitle())
						user.MarkMessageSent(header)
					} else {
						switch err.(type) {
						case mwclient.APIError:
							switch err.(mwclient.APIError).Code {
							case "noedit", "writeapidenied", "blocked":
								log.Fatal("noedit/writeapidenied/blocked code returned, the bot may have been blocked. Dying")
							case "pagedeleted":
								log.Println("Looks like the user", user.Username, "talk page was deleted while we were updating it... huh. Going for a new one!")
							default:
								log.Println("Error editing user talk for", user.Username, "meant they couldn't be notified and were ignored. The error was", err)
							}
						default:
							log.Fatal("Non-API error returned when trying to notify user ", user.Username, " so dying. Error was ", err)
						}
					}
				}
			}
		}
	} else {
		log.Println("WARNING: Headers to send to returned as less than one for page", requester.PageTitle(), "so ignoring for now, but this could be a bug")
	}
}
