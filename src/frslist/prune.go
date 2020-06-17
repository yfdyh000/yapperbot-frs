package frslist

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	// needs to be blank-imported to make the driver work
	_ "github.com/go-sql-driver/mysql"
)

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

var checkedUsers map[string]bool = map[string]bool{}
var usersToRemove [][]string
var usersToReplace map[string]string = map[string]string{}

const lastEditQueryTemplate string = `SELECT actor_user.actor_name FROM revision_userindex
INNER JOIN actor_user ON actor_user.actor_name = ? AND actor_id = rev_actor
WHERE rev_timestamp > ? LIMIT 1;`

const blockQueryTemplate string = `SELECT ipb_id FROM ipblocks
INNER JOIN user ON user_name = ? AND user_id = ipb_user
WHERE ipb_expiry = "infinity" LIMIT 1;`

const userRedirectQueryTemplate string = `SELECT rd_title FROM redirect
INNER JOIN page ON page_namespace = 3 AND page_title = ? AND rd_from = page_id
WHERE page_is_redirect = 1 AND rd_namespace = 3 LIMIT 1;`

func pruneUsersFromList(text string, dbserver, dbuser, dbpassword, db string) {
	var regexBuilder strings.Builder

	conn, err := sql.Open("mysql", dbuser+":"+dbpassword+"@tcp("+dbserver+")/"+db)
	if err != nil {
		log.Fatal("DSN invalid with error ", err)
	}
	if err := conn.Ping(); err != nil {
		log.Fatal(err)
	}

	lastEditQuery, err := conn.Prepare(lastEditQueryTemplate)
	if err != nil {
		log.Fatal("lastEditQuery preparation failed with error ", err)
	}
	defer lastEditQuery.Close()

	blockQuery, err := conn.Prepare(blockQueryTemplate)
	if err != nil {
		log.Fatal("blockQuery preparation failed with error ", err)
	}
	defer blockQuery.Close()

	userRedirectQuery, err := conn.Prepare(userRedirectQueryTemplate)
	if err != nil {
		log.Fatal("userRedirectQuery preparation failed with error ", err)
	}
	defer userRedirectQuery.Close()

	var editsSinceStamp string = time.Now().AddDate(-3, 0, 0).Format("20060102210405") // format in line with https://www.mediawiki.org/wiki/Manual:Timestamp

	for _, header := range list {
		for _, user := range header {
			var outputFromQueryRow string

			if checkedUsers[user.Username] {
				continue
			}

			checkedUsers[user.Username] = true
			var dbUsername string = usernameCase(user.Username)
			// We have no use whatsoever for the output of this, we just want to see if it errors.
			// That being said, Scan() doesn't let us just pass nothing, so we have to have the
			// slight pain of having a stupid additional variable.
			err := lastEditQuery.QueryRow(dbUsername, editsSinceStamp).Scan(&outputFromQueryRow)

			if err != nil {
				if err == sql.ErrNoRows {
					// they haven't edited in the timeframe, or they have redirected
					// check a redirect for them
					err := userRedirectQuery.QueryRow(dbUsername).Scan(&outputFromQueryRow)
					if err == sql.ErrNoRows {
						// No redirect found, just remove them
						log.Println("Queuing", user.Username, "for pruning")
						usersToRemove = append(usersToRemove, []string{user.Username, "timeout"})
						continue
					} else if err != nil {
						log.Fatal("Failed when querying DB for redirects with error ", err)
					}
					// A redirect was found!
					outputFromQueryRow = strings.ReplaceAll(outputFromQueryRow, "_", " ")
					log.Println("Found a redirect for", user.Username, "so replacing them with", outputFromQueryRow)
					usersToReplace[user.Username] = outputFromQueryRow
					// this is here to make sure that the redirect target is also checked for indefs
					user.Username = outputFromQueryRow
				} else {
					log.Fatal("Failed when querying DB for last edits with error ", err)
				}
			}

			// if they still aren't being pruned, check whether they're indefinitely blocked
			err = blockQuery.QueryRow(dbUsername).Scan(&outputFromQueryRow)
			if err == nil {
				// the user is indeffed, as a row has been found
				log.Println("Queuing indeffed user", user.Username, "for pruning")
				usersToRemove = append(usersToRemove, []string{user.Username, "indeffed"})
				continue
			} else if err != sql.ErrNoRows {
				log.Fatal("Failed when querying DB for blocks with error ", err)
			}
		}
	}

	regexBuilder.WriteString(`(?i)\* ?{{frs user\|(`) // Write the start of the regex
	regexUsersToRemove := make([]string, len(usersToRemove))
	for i, user := range usersToRemove {
		regexUsersToRemove[i] = regexp.QuoteMeta(user[0])
	}
	regexBuilder.WriteString(strings.Join(regexUsersToRemove, "|"))
	regexBuilder.WriteString(`)\|\d+}}\n?`)

	usersRemovedInfo := make([]string, len(usersToRemove))
	for i, user := range usersToRemove {
		usersRemovedInfo[i] = strings.Join(user, ": ")
	}

	var renamedUsersBuilder strings.Builder
	for old, new := range usersToReplace {
		// replace all parameter instances (all ought to be between pipes) with the new username
		text = strings.ReplaceAll(text, "|"+old+"|", "|"+new+"|")
		renamedUsersBuilder.WriteString(old)
		renamedUsersBuilder.WriteString(" -> ")
		renamedUsersBuilder.WriteString(new)
		renamedUsersBuilder.WriteString("\n")
	}

	fmt.Println("=========================================================================================")
	fmt.Println("======================================= WIKITEXT ========================================")
	fmt.Println("=========================================================================================")
	fmt.Println(regexp.MustCompile(regexBuilder.String()).ReplaceAllString(text, ""))
	fmt.Println("=========================================================================================")
	fmt.Println("===================================== REMOVED USERS =====================================")
	fmt.Println("=========================================================================================")
	fmt.Println(strings.Join(usersRemovedInfo, "\n"))
	fmt.Println("=========================================================================================")
	fmt.Println("===================================== RENAMED USERS =====================================")
	fmt.Println("=========================================================================================")
	fmt.Println(renamedUsersBuilder.String())
}

// Takes a string, s, and converts the first rune to uppercase
// by using the unicode functions in golang's stdlib
// Returns the converted string
func usernameCase(s string) string {
	firstRune, size := utf8.DecodeRuneInString(s)
	if firstRune != utf8.RuneError || size > 1 {
		var upcase rune

		if replacement, exists := charReplaceUpcase[firstRune]; exists {
			upcase = replacement
		} else {
			upcase = unicode.ToUpper(firstRune)
		}
		if upcase != firstRune {
			s = string(upcase) + s[size:]
		}
	}
	return strings.ReplaceAll(s, "_", " ")
}
