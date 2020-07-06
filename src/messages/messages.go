// Package messages contains our message queueing and sending functionality.
package messages

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
	"strconv"
	"strings"
	"time"
	"yapperbot-frs/src/frslist"

	"cgt.name/pkg/go-mwclient"
	"cgt.name/pkg/go-mwclient/params"
	"github.com/mashedkeyboard/ybtools/v2"
)

// A Message object represents a single message which might be sent to a user.
type Message struct {
	// This User is deliberately on the Message, and is not the key of messagesToSend,
	// because a FRSUser represents a user's subscription, not the user itself. A user
	// may have multiple FRSUser objects if they are subscribed to multiple headers.
	User *frslist.FRSUser
	Type string
	// Title refers to the title of the page the message is about, not the message title.
	Title string
	RFCID string
}

// messagesToSend is our username-indexed list of messages that we have queued.
// Each username key maps to a list of messages we have stored up to send them this run.
var messagesToSend = map[string][]*Message{}

// commentRegex matches HTML comments, allowing us to remove them;
// we use it to clean our headers before we send to users.
var commentRegex *regexp.Regexp

// cleanedHeaders is a map mapping our "dirty" headers (those containing
// the HTML comments) to cleaned versions, that have had comments removed
// using the commentRegex.
var cleanedHeaders = map[string]string{}

// editSummaryForFeedbackMsgs is used to generate our edit summary. We run Sprintf over it
// with the appropriately-formatted values we get back from editSummaryMessagesComponent, joined together with
// a limitInEditSummary formatted as necessary if the user has a limit set for the category.
const editSummaryForFeedbackMsgs string = `[[WP:FRS|Feedback Request Service]] notification on %s. You can unsubscribe at [[WP:FRS]].`

// editSummaryMessagesComponent contains the core part of our edit summary. We run Sprintf over it with:
// %s 1: header the user was subscribed to
// %s 2: the type of request (GA nom, RfC, etc)
// %s 3: limitInEditSummary, or empty string for no limit
const editSummaryMessagesComponent string = `a "%s" %s%s`

// limitInEditSummary is used where users have a limit set.
// Sprintf is run over it with the first param as the used amount, and the second as the limit.
const limitInEditSummary string = ` (%d/%d this month)`

func init() {
	commentRegex = regexp.MustCompile(`\s*?<!--.*?-->\s*?`)
}

// QueueMessage takes a pointer to a Message, and adds it into our queue
// of messages to send to this user once we've finished our run and we're actually
// sending the messages that we've processed.
func QueueMessage(m *Message) {
	messagesToSend[m.User.Username] = append(messagesToSend[m.User.Username], m)
}

// SendMessageQueue takes a pointer to an mwclient instance, and sends all the queued
// messages from the FRS run.
func SendMessageQueue(w *mwclient.Client) {
	for user, messages := range messagesToSend {
		var textBuilder *strings.Builder
		var summarySentListBuilder strings.Builder

		textBuilder.WriteString("{{subst:User:Yapperbot/FRS notification")

		for index, message := range messages {
			strindex := strconv.Itoa(index)
			numberedParamToBuilder(textBuilder, strindex, "title")
			textBuilder.WriteString(message.Title)
			numberedParamToBuilder(textBuilder, strindex, "type")
			textBuilder.WriteString(message.Type)
			if message.RFCID != "" {
				numberedParamToBuilder(textBuilder, strindex, "rfcid")
				textBuilder.WriteString(message.RFCID)
			}

			var limitsummary string
			if message.User.Limited {
				limitsummary = fmt.Sprintf(limitInEditSummary, message.User.GetCount()+1, message.User.Limit)
			}
			summarySentListBuilder.WriteString(fmt.Sprintf(editSummaryMessagesComponent, message.User.Header, message.Type, limitsummary))

			if len(messages) > 1 && index != len(messages)-1 {
				summarySentListBuilder.WriteString(", ")
				if index == len(messages)-2 {
					summarySentListBuilder.WriteString("and ")
				}
			}
		}

		textBuilder.WriteString("}} ~~~~")
		var notificationText string = textBuilder.String()

		var sectiontitle string
		if len(messages) == 1 {
			cleanedHeader := cleanedHeaders[messages[0].User.Header]
			sectiontitle = fmt.Sprintf("Feedback request: %s %s", cleanedHeader, messages[0].Type)
		} else {
			sectiontitle = "Feedback requests from the Feedback Request Service"
		}

		// Drop a note on each user's talk page inviting them to participate
		if ybtools.CanEdit() {
			// Generate the edit summary, with their limit
			editsummary := fmt.Sprintf(editSummaryForFeedbackMsgs, summarySentListBuilder.String())

			// the redirect param here automatically resolves redirects,
			// for instance if a user changes their username but forgets
			// to update the FRS user tag
			err := w.Edit(params.Values{
				"title":        "User talk:" + user,
				"section":      "new",
				"sectiontitle": sectiontitle,
				"summary":      editsummary,
				"notminor":     "true",
				"bot":          "true",
				"text":         notificationText,
				"redirect":     "true",
			})
			if err == nil {
				log.Println("Successfully invited", user, "to give feedback on", len(messages), "requesting items")
				for _, message := range messages {
					message.User.MarkMessageSent()
				}
				time.Sleep(5 * time.Second)
			} else {
				switch err.(type) {
				case mwclient.APIError:
					switch err.(mwclient.APIError).Code {
					case "noedit", "writeapidenied", "blocked":
						ybtools.PanicErr("noedit/writeapidenied/blocked code returned, the bot may have been blocked. Dying")
					case "pagedeleted":
						log.Println("Looks like the user", user, "talk page was deleted while we were updating it... huh. Going for a new one!")
					default:
						log.Println("Error editing user talk for", user, "meant they couldn't be notified and were ignored. The error was", err)
					}
				default:
					ybtools.PanicErr("Non-API error returned when trying to notify user ", user, " so dying. Error was ", err)
				}
			}
		}
	}
}

// CleanHeader takes a "dirty" header (a header with HTML comments in) as a string,
// cleans it up, and saves it into our processed headers in cleanedHeaders. This is
// used so that we don't end up sending HTML comments to users, which aren't very pretty!
func CleanHeader(header string) {
	// check if we've already cleaned the header previously
	if _, ok := cleanedHeaders[header]; !ok {
		// we've not done it previously!
		// clean the header and save it here, so we don't have to run a regex on every user
		cleanedHeaders[header] = commentRegex.ReplaceAllString(header, "")
	}
}

// numberedParamToBuilder takes a strings.Builder, an index as a string,
// and a parameter name to go along with that index,
// and adds the relevant bits for a MediaWiki parameter to the builder.
func numberedParamToBuilder(b *strings.Builder, i, param string) {
	b.WriteString("|")
	b.WriteString(param)
	b.WriteString(i)
	b.WriteString("=")
}
