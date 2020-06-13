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

	"cgt.name/pkg/go-mwclient"

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

var checkedUsers map[string]bool
var usersToRemove []string
var regexEscapeNeeded *regexp.Regexp

const queryTemplate string = `SELECT actor_user.actor_name FROM revision_userindex
INNER JOIN actor_user ON actor_user.actor_name = ? AND actor_id = rev_actor
WHERE rev_timestamp > ? LIMIT 1;`

func init() {
	regexEscapeNeeded = regexp.MustCompile(`[.*+?^${}()|[\]\\]`)
	checkedUsers = map[string]bool{}
}

func pruneUsersFromList(text string, w *mwclient.Client, dbserver, dbuser, dbpassword, db string) {
	var regexBuilder strings.Builder
	var ignoredOutputFromQueryRow string

	conn, err := sql.Open("mysql", dbuser+":"+dbpassword+"@tcp("+dbserver+")/"+db)
	if err != nil {
		log.Fatal("DSN invalid with error ", err)
	}
	if err := conn.Ping(); err != nil {
		log.Fatal(err)
	}
	query, err := conn.Prepare(queryTemplate)
	if err != nil {
		log.Fatal(err)
	}
	defer query.Close()

	var editsSinceStamp string = time.Now().AddDate(-3, 0, 0).Format("20060102210405") // format in line with https://www.mediawiki.org/wiki/Manual:Timestamp

	for _, header := range list {
		for _, user := range header {
			if checkedUsers[user.Username] {
				continue
			}

			checkedUsers[user.Username] = true
			// We have no use whatsoever for the output of this, we just want to see if it errors.
			// That being said, Scan() doesn't let us just pass nothing, so we have to have the
			// slight pain of having a stupid additional variable.
			err := query.QueryRow(usernameCase(user.Username), editsSinceStamp).Scan(&ignoredOutputFromQueryRow)
			if err != nil {
				if err == sql.ErrNoRows {
					log.Println("Queuing", user.Username, "for pruning")
					usersToRemove = append(usersToRemove, user.Username)
				} else {
					log.Fatal("Failed when querying DB with error ", err)
				}
			}
		}
	}

	regexBuilder.WriteString(`(?i)\* ?{{frs user\|(`) // Write the start of the regex
	regexUsersToRemove := make([]string, len(usersToRemove))
	for i, user := range usersToRemove {
		regexUsersToRemove[i] = regexEscapeNeeded.ReplaceAllString(user, `\$0`)
	}
	regexBuilder.WriteString(strings.Join(regexUsersToRemove, "|"))
	regexBuilder.WriteString(`)\|(\d+)}}\n?`)

	fmt.Println("=========================================================================================")
	fmt.Println("======================================= WIKITEXT ========================================")
	fmt.Println("=========================================================================================")
	fmt.Println(regexp.MustCompile(regexBuilder.String()).ReplaceAllString(text, ""))
	fmt.Println("=========================================================================================")
	fmt.Println("===================================== REMOVED USERS =====================================")
	fmt.Println("=========================================================================================")
	fmt.Println(strings.Join(usersToRemove, "\n"))
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
