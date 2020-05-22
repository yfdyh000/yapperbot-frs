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
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
	"yapperbot-frs/src/frslist"
	"yapperbot-frs/src/ga"
	"yapperbot-frs/src/rfc"
	"yapperbot-frs/src/yapperconfig"

	"cgt.name/pkg/go-mwclient"
	"cgt.name/pkg/go-mwclient/params"
	"github.com/mashedkeyboard/ybtools"
	"github.com/metal3d/go-slugify"
)

func main() {
	var w *mwclient.Client
	var err error

	rand.Seed(time.Now().UnixNano())

	w, err = mwclient.New(yapperconfig.Config.APIEndpoint, "Yapperbot-FRS v1")
	if err != nil {
		log.Fatal("Failed to create MediaWiki client with error ", err)
	}

	err = w.Login(yapperconfig.Config.BotUsername, os.Getenv("WP_BOT_PASSWORD"))
	if err != nil {
		log.Fatal("Failed to authenticate with MediaWiki with error ", err)
	}

	frslist.Populate(w)
	defer frslist.FinishRun(w)
	defer yapperconfig.SaveEditLimit()

	ga.FetchGATopics(w)

	queryCategory(w, "Category:Wikipedia requests for comment", true)
	queryCategory(w, "Category:Good article nominees", false)
}

func queryCategory(w *mwclient.Client, category string, rfcCat bool) {
	var startStamp, startID string
	var newRunfile bool
	if !rfcCat {
		startStamp, startID = loadFromRunfile(category)
		if startStamp == "" {
			startStamp = time.Now().Format(time.RFC3339)
			// Set our runfile to store this now, as there's potentially going to be nothing in the queue
			newRunfile = true
		}
	}

	var parameters params.Values

	if rfcCat {
		// gets a list of all active RfCs. We'll manage which ones to deal with later
		parameters = params.Values{
			"action":    "query",
			"prop":      "revisions",
			"generator": "embeddedin",
			"geititle":  "Template:Rfc",
			"rvprop":    "content",
			"rvslots":   "main",
		}
	} else {
		parameters = params.Values{
			"action":       "query",
			"prop":         "revisions|categories",
			"generator":    "categorymembers",
			"gcmtitle":     category,
			"gcmsort":      "timestamp",
			"rvprop":       "content",
			"rvslots":      "main",
			"clprop":       "timestamp",
			"clcategories": category,
			"gcmdir":       "descending",
			"gcmstart":     time.Now().Add(-time.Hour).Format(time.RFC3339), // give it at least an hour of tranquility before invites go out
			"gcmend":       startStamp,                                      // this is gcmend not gcmstart as it's going down from the most recent
		}
	}

	var firstItem string = ""
	query := w.NewQuery(parameters)

	for query.Next() {
		pages := ybtools.GetPagesFromQuery(query.Resp())
		if len(pages) > 0 {
			if !rfcCat {
				// on the first item of the entire set, and the first item ONLY, save the timestamp and the page id into a var to write to runfile later
				// RfCs don't use this as there's fewer of them and it's more difficult to get them in timestamp order
				if len(firstItem) == 0 {
					var runfileBuilder strings.Builder
					runfileBuilder.WriteString(ybtools.GetCategorisationTimestampFromPage(pages[0], category))
					runfileBuilder.WriteString(";")

					firstItemPageID, err := pages[0].GetInt64("pageid")
					if err != nil {
						log.Fatal("Failed to get pageid from the first item in the queue with error message ", err)
					}
					runfileBuilder.WriteString(string(firstItemPageID))
					firstItem = runfileBuilder.String()
				}
			}

		PAGELOOP:
			for index, page := range pages {
				pageIDInt, err := page.GetInt64("pageid")
				if err != nil {
					log.Fatal("Failed to get pageid from page in category ", category, " with index ", index, ", error was: ", err)
				}
				pageID := strconv.FormatInt(pageIDInt, 10) // format it into a string integer

				pageTitle, err := page.GetString("title")
				if err != nil {
					log.Println("Failed to get title from page ID", pageID, "so skipping it")
					continue
				}

				pageContent, err := ybtools.GetContentFromPage(page)
				if err != nil {
					log.Println("getContentFromPage failed on page ID", pageID, "so skipping it")
					continue
				}

				if rfcCat {
					// (content, title, excludeDone)
					rfcsToProcess, err := extractRfcs(pageContent, pageTitle, true)
					if err != nil {
						if _, isRfCNoIDYet := err.(rfc.NoRfCIDYetError); isRfCNoIDYet {
							// if any of the RfCs don't yet have an ID, we skip the page - it'll be included in the next run as this is an RfC page
							log.Println("RfC has no ID yet on page", pageTitle, "so skipping")
							continue PAGELOOP
						} else {
							log.Fatal("extractRfcs errored with ", err)
						}
					}
					rfcsDone := make([]rfc.RfC, 0, len(rfcsToProcess))

				RFCLOOP:
					for _, rfc := range rfcsToProcess {
						if rfc.FeedbackDone {
							log.Println("RfC feedback already done for an RfC on", pageTitle, "so skipping")
							continue RFCLOOP
						} else {
							log.Println("Requesting feedback for an RfC on", pageTitle)
							requestFeedbackFor(rfc, w)
							rfcsDone = append(rfcsDone, rfc)
						}
					}
					if len(rfcsDone) > 0 {
						rfc.MarkRfcsDone(w, pageID, rfcsDone)
					}
				} else {
					// Because each article can only have one GA nomination at a time, it's not necessary to do the full gamut of RfC checks here
					// we can instead just pass it on to requestFeedbackFor after checking that it's not the same page we did first last time
					// to do that check, we check whether the page ID and timestamp are the same (both stored in the runfile) - if they are, it's the same page
					if (pageID == startID) && (ybtools.GetCategorisationTimestampFromPage(page, category) == startStamp) {
						// it's the first page from last time, we're probably at the end - skip over it
						continue PAGELOOP
					} else {
						requestFeedbackFor(extractGANom(pageContent, pageTitle), w)
					}
				}
			}
		} else if newRunfile && len(pages) == 0 {
			// if it's a new file and no pages are picked up, just create the runfile so future runs will know where to start from
			log.Println("No pages found, and a new runfile, so creating runfile with current time for", category)
			firstItem = startStamp + ";"
		}
	}
	if query.Err() == nil {
		log.Println("Finished the queue for category", category, "so ending here")
	} else {
		log.Fatal("Errored while querying for relevant new pages with error: ", query.Err())
	}

	// If it uses a runfile, and there actually is something to write
	if !rfcCat && len(firstItem) > 0 {
		// Store the done timestamp and page id into the runfile for next use
		err := ioutil.WriteFile(slugify.Marshal(category)+".frsrunfile", []byte(firstItem), 0644)
		if err != nil {
			log.Fatal("Failed to write timestamp and id to runfile")
		}
	}
}
