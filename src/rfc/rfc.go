package rfc

import "regexp"

var rfcPrefixRegex *regexp.Regexp

const requestType string = "request for comment"

// An RfC has an id, a categories map and a setting for whether feedback has been given for it.
// The map should be map[string]bool, with bool as true for every element
// This is so membership verification is o(1) rather than o(n)
type RfC struct {
	ID           string
	Categories   map[string]bool
	FeedbackDone bool
	PageHolding  string
}

func init() {
	rfcPrefixRegex = regexp.MustCompile(`<!--rfc:(\w*?)-->`)
}

// IncludeHeader determines if a given FRS header corresponds to this item correctly
// Takes a string of the entire header (minus the === bits) and returns true or false
func (r RfC) IncludeHeader(header string) bool {
	matches := rfcPrefixRegex.FindStringSubmatch(header)
	if matches == nil {
		// no matches means it's not an RfC
		return false
	}

	// check for special keyword "all"
	if matches[1] == "all" {
		return true
	}
	// check if in categories
	_, exists := r.Categories[matches[1]]
	return exists
}

// PageTitle is a simple getter for the HoldingPage in order to make the interface work
func (r RfC) PageTitle() string {
	return r.PageHolding
}

// RequestType returns the type this is - an RfC - so that it can be used in a template
func (r RfC) RequestType() string {
	return requestType
}
