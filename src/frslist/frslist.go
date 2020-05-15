package frslist

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
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"yapperbot-frs/src/wikinteract"
	"yapperbot-frs/src/yapperconfig"

	"cgt.name/pkg/go-mwclient"
	"cgt.name/pkg/go-mwclient/params"
	"github.com/antonholmquist/jason"
)

// list is the overall list of FRSUsers mapped to their headers.
// listHeaders is just a boring old list of headers, we have a getter for it later.
var list map[string][]FRSUser
var listHeaders []string

// sentCount maps headers down to users, and then users down to the number of messages they've received this month.
// the Mux is just a mux for it in case the app gets goroutines at some point.
var sentCount map[string]map[string]int16 // {header: {user: count sent}}
var sentCountMux sync.Mutex

var listParserRegex *regexp.Regexp
var userParserRegex *regexp.Regexp

var randomGenerator *rand.Rand

var frsPageID string = yapperconfig.Config.FRSPageID
var sentCountPageID string = yapperconfig.Config.SentCountPageID

func init() {
	// This regex matches on the Feedback Request Service list.
	// The first group matches the header (minus the ===s)
	// The second matches all of the contents underneath that header
	listParserRegex = regexp.MustCompile(`===(.*?)===\n((?i:\*\s*{{frs user.*?}}\n?)+)`)

	// This regex matches each user individually in a section of the FRS list.
	// The first group matches the user name
	// The second group matches the requested limit
	userParserRegex = regexp.MustCompile(`(?i){{frs user\|(.*)\|(\d+)}}`)

	randomGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))
	list = map[string][]FRSUser{}
	sentCount = map[string]map[string]int16{}
}

// Populate sets up the FRSList list as appropriate for the start of the program.
func Populate(w *mwclient.Client) {
	populateFrsList(w)
	populateSentCount(w)
}

// GetListHeaders is a simple getter for listHeaders
func GetListHeaders() []string {
	return listHeaders
}

// GetUsersFromHeaders takes a list of headers and an integer number of users n, and returns a randomly selected portion of the users
// from each header, with each header of size n. It won't pick the same user twice.
func GetUsersFromHeaders(headers []string, n int) (headerusers map[string][]FRSUser) {
	headerusers = make(map[string][]FRSUser)
	pickedNumbers := map[int]bool{}

	for _, header := range headers {
		users := make([]FRSUser, 0, n)

		i := 0
		for i < n {
			var user FRSUser

			if len(list[header]) <= n {
				// very small list, or very large n
				// just give the entire list after checking for user limits

				if i >= len(list[header]) {
					// if we've looped over the entire length of the header list, break out to the main header loop
					break
				} else if pickedNumbers[i] {
					// if the user is already included, skip it and increment i here - we're not randomly selecting, we don't have that many users to choose from
					i++
					continue
				}

				pickedNumbers[i] = true
				user = list[header][i]
			} else {
				selected := randomGenerator.Intn(len(list[header]))
				if pickedNumbers[selected] {
					// the number has already been picked, do it again without incrementing i
					continue
				}
				// number not yet picked, now we add the number to the picked list and see if the user is valid
				pickedNumbers[selected] = true
				user = list[header][selected]
			}

			if user.GetCount(header) >= user.Limit {
				// user has exceeded limit, or this message would cause them to exceed the limit; ignore them and move on
				continue
			}

			// user is good to go! expand the slice...
			users = users[:len(users)+1]
			// ... and add them to the list!
			users[i] = user
			i++
		}

		headerusers[header] = users
	}

	return
}

// FinishRun for now just calls saveSentCounts, but could do something else too in future
func FinishRun(w *mwclient.Client) {
	saveSentCounts(w)
}

func populateFrsList(w *mwclient.Client) {
	text, err := wikinteract.FetchWikitext(w, frsPageID)
	if err != nil {
		log.Fatal("Failed to fetch and parse FRS page with error ", err)
	}

	for _, match := range listParserRegex.FindAllStringSubmatch(text, -1) {
		// match is [entire match, header, contents]
		var users []FRSUser
		for _, usermatched := range userParserRegex.FindAllStringSubmatch(match[2], -1) {
			// usermatched is [entire match, user name, requested limit]
			if limit, err := strconv.ParseInt(usermatched[2], 10, 16); err == nil {
				users = append(users, FRSUser{usermatched[1], int16(limit)})
			} else {
				log.Println("User", usermatched[1], "has an invalid limit of", usermatched[2], "so ignoring")
			}
		}
		list[match[1]] = users
	}

	listHeaders = make([]string, len(list))
	i := 0
	for header := range list {
		listHeaders[i] = header
		i++
	}
}

func populateSentCount(w *mwclient.Client) {
	// This is stored on the page with ID sentCountPageID.
	// It is made up of something that looks like this:
	// {"month": "2020-05", "headers": {"category": {"username": 8}}}
	// where username had been sent 8 messages in the month of May 2020 and the header "category".

	storedJSON, err := wikinteract.FetchWikitext(w, sentCountPageID)
	if err != nil {
		log.Fatal("Failed to fetch sent count page with error ", err)
	}
	parsedJSON, err := jason.NewObjectFromBytes([]byte(storedJSON))
	if err != nil {
		log.Fatal("Failed to parse sent count JSON with error ", err)
	}

	contentMonth, _ := parsedJSON.GetString("month")
	// yes, really, you have to specify time formats with a specific time in Go
	// *rolls eyes*
	// https://golang.org/pkg/time/#Time.Format
	if contentMonth != time.Now().Format("2006-01") {
		log.Println("contentMonth is not the current month, so data resets!")
	} else {
		sentCount = deserializeSentCount(parsedJSON)
	}
}

func saveSentCounts(w *mwclient.Client) {
	var sentCountJSONBuilder strings.Builder
	sentCountJSONBuilder.WriteString(`{"DO NOT TOUCH THIS PAGE":"This page is used internally by Yapperbot to make the Feedback Request Service work.","month":"`)
	sentCountJSONBuilder.WriteString(time.Now().Format("2006-01"))
	sentCountJSONBuilder.WriteString(`","headers":`)
	sentCountJSONBuilder.WriteString(serializeSentCount(sentCount))
	sentCountJSONBuilder.WriteString(`}`)

	err := w.Edit(params.Values{
		"pageid":   sentCountPageID,
		"summary":  "FRS run complete, updating sentcounts",
		"notminor": "true",
		"bot":      "true",
		"text":     sentCountJSONBuilder.String(),
	})
	if err == nil {
		log.Println("Successfully updated sentcounts")
	} else {
		if err.Error() == "edit successful, but did not change page" {
			log.Println("WARNING: Successfully updated sentcounts, but they didn't change - if anything was done this session, something is wrong!")
		} else {
			log.Fatal("Failed to update sentcounts with error ", err)
		}
	}
}
