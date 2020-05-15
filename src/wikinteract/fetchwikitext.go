package wikinteract

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
	"cgt.name/pkg/go-mwclient"
	"cgt.name/pkg/go-mwclient/params"
)

// FetchWikitext takes a client and a pageId and gets the wikitext of that page.
// The default functionality in the library does not work for this in
// my experience; it just returns an empty string for some reason. So we're rolling our own!
func FetchWikitext(w *mwclient.Client, pageID string) (content string, err error) {
	pageContent, err := w.Get(params.Values{
		"action": "parse",
		"pageid": pageID,
		"prop":   "wikitext",
	})
	if err != nil {
		return "", err
	}
	text, err := pageContent.GetString("parse", "wikitext")
	if err != nil {
		return "", err
	}
	return text, nil
}
