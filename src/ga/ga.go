package ga

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
	"strings"
)

const gaPrefix string = "<!--gan-->"
const requestType string = "Good Article nomination"

// Nom represents a GA nomination, which has a single category only.
type Nom struct {
	Topic    string
	Article  string
	Subtopic string
}

// IncludeHeader determines if a given FRS header corresponds to this item correctly
// Takes a string of the entire header (minus the === bits) and returns true or false
func (n Nom) IncludeHeader(header string) bool {
	// TrimPrefix does nothing if the prefix isn't there, so this is fine
	headerSansPrefix := strings.TrimPrefix(header, gaPrefix)
	// if it's the topic, or the subtopic's respective topic from a gaTopics lookup
	if headerSansPrefix == n.Topic || (gaTopics[n.Subtopic] != "" && headerSansPrefix == gaTopics[n.Subtopic]) {
		return true
	}
	return false
}

// PageTitle is a simple getter for the GA nominee article in order to make the interface work
func (n Nom) PageTitle() string {
	return n.Article
}

// RequestType returns the type this is - a GA nom - so that it can be used in a template
func (n Nom) RequestType() string {
	return requestType
}
