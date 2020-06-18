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

// An FRSUser is a struct representing a user who has signed up for the FRS.
// A single username may have multiple FRSUser objects; each corresponds to an
// individual subscription.
type FRSUser struct {
	Username string
	Limit    int16
	Limited  bool
}

// GetCount takes a header and gets the number of messages sent for that header this month.
func (f FRSUser) GetCount(header string) int16 {
	sentCountMux.Lock()
	defer sentCountMux.Unlock()
	return sentCount[header][f.Username]
}

// ExceedsLimit is a simple helper function for checking if a user is limited,
// and if they are, whether they can be messaged according to their limits.
func (f FRSUser) ExceedsLimit(header string) bool {
	if f.Limited {
		return (f.GetCount(header) >= f.Limit)
	}
	return false
}

// MarkMessageSent takes a header and increases the number of messages sent for that header by one.
func (f FRSUser) MarkMessageSent(header string) {
	sentCountMux.Lock()
	defer sentCountMux.Unlock()

	// prevent nil map errors
	if sentCount[header] == nil {
		sentCount[header] = map[string]int16{}
	}

	sentCount[header][f.Username]++
}
