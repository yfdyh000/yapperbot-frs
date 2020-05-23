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
	"yapperbot-frs/src/yapperconfig"

	"cgt.name/pkg/go-mwclient"
	"cgt.name/pkg/go-mwclient/params"
	"github.com/antonholmquist/jason"
	"github.com/mashedkeyboard/ybtools"
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
	listParserRegex = regexp.MustCompile(`===(.*?)===\n((?i:\*\s*{{frs user.*?}}\n*)+)`)

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
	// maps header to array of users
	headerusers = map[string][]FRSUser{}
	// maps user to true if used - used for o(1) lookups of the user to check if already included under any header
	pickedusers := map[string]bool{}

	for _, header := range headers {
		users := make([]FRSUser, 0, n)

		if len(list[header]) <= n {
			// very small list, or very large n
			// just give the entire list after checking for user limits
			for _, user := range list[header] {
				// what happens with this doesn't matter here, as we're literally just adding all qualifying users
				checkUserAndIncludeInHeader(user, &pickedusers, header, &users)
			}
		} else {
			// get random indexes (.Perm returns a random permutation of 0-(n-1))
			for _, i := range randomGenerator.Perm(len(list[header])) {
				if len(users) >= n {
					// we've already picked the number requested, stop picking
					break
				}
				checkUserAndIncludeInHeader(list[header][i], &pickedusers, header, &users)
			}
		}

		headerusers[header] = users
	}

	return
}

// takes a user, a pickedusers map, the header, and the list of users
// checks if the user is eligible for inclusion and if they are, adds them to pickedusers and users
func checkUserAndIncludeInHeader(user FRSUser, pickedusers *map[string]bool, header string, users *[]FRSUser) {
	if (*pickedusers)[user.Username] {
		// if the user is already included, skip it
		return
	}

	(*pickedusers)[user.Username] = true

	if user.GetCount(header) >= user.Limit {
		// user has exceeded limit, or this message would cause them to exceed the limit; ignore them and move on
		return
	}

	// user is good to go! expand the slice...
	// expanding the slice in here means we need a pointer to a slice, not just the slice.
	// if it was just the slice, it would update the elements in the underlying array; however,
	// we're changing the length, which means changing the slice itself, which needs a pointer
	oldlen := len(*users)
	*users = (*users)[:oldlen+1] // oldlen+1 expands the length by 1, the slice notation here uses length, not index
	// ... and add them to the list! (oldlen is now the last key of the new slice)
	(*users)[oldlen] = user
}

// FinishRun for now just calls saveSentCounts, but could do something else too in future
func FinishRun(w *mwclient.Client) {
	saveSentCounts(w)
}

func populateFrsList(w *mwclient.Client) {
	text, err := ybtools.FetchWikitext(w, frsPageID)
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

	storedJSON, err := ybtools.FetchWikitext(w, sentCountPageID)
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

	// this is in userspace, and it's really desperately necessary - do not count this for edit limiting
	// if yapperconfig.EditLimit() {
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
	// }
}
