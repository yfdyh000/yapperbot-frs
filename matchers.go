package main

import (
	"regexp"
	"strings"
	"yapperbot-frs/src/ga"
	"yapperbot-frs/src/rfc"
)

var rfcMatcher *regexp.Regexp
var gaMatcher *regexp.Regexp
var namedParamMatcher *regexp.Regexp

const rfcIDParam string = "rfcid="
const rfcFeedbackDoneParam string = "frsdone="

func init() {
	// RfC matching regex.
	// First capture group is all the params of the rfc template
	// (all non-named params are to be treated as categories)
	// second capture group is the actual content of the RfC
	rfcMatcher = regexp.MustCompile(`(?i){{rfc\|(.*?)}}(.|\n)*?\(UTC\)`)

	// GA nom matching regex.
	// First capture group is the topic. Second capture group is the subtopic.
	// IMPORTANT: If the topic is empty, the first capture group will be empty string, and likewise for the subtopic.
	// This regex will always prefer a topic to a subtopic, but will settle for a subtopic if no topic is available.
	// Great thanks go to Ouims from #regex on Freenode for the help with debugging and correcting this regex!
	gaMatcher = regexp.MustCompile(`(?i){{GA nominee(?:\|(?:[^|}]*?\|)*(?:[\t\f\v ]*?(?:subtopic=([^|}]+).*?)|topic=([^|}]+))|.*?)*}}`)

	// Matches against named parameters in the parameter list.
	// Ensures the equals is after a named param specifically.
	// The [\w\d\s] set means it won't trigger on {{=}} and the like.
	namedParamMatcher = regexp.MustCompile(`(?m)^[\w\d\s]*?=`)
}

// extractRfcs takes a string of content containing rfcs, and the page title,
// and returns a slice of rfcs. It can optionally be passed excludeDone, which prevents
// already-done RfCs from being included in the generated list.
func extractRfcs(content string, title string, excludeDone bool) (rfcs []rfc.RfC, err error) {
	matchedRfcTags := rfcMatcher.FindAllStringSubmatch(content, -1)
	for _, tag := range matchedRfcTags {
		// group 1 contains all the parameters; split on | to find the individual params
		params := strings.Split(tag[1], "|")

		rfcID, categories, feedbackDone := func(pr []string) (id string, cats map[string]bool, done bool) {
			cats = make(map[string]bool)
			for _, p := range pr {
				if strings.Contains(p, rfcIDParam) {
					id = strings.TrimPrefix(p, rfcIDParam)
				} else if strings.Contains(p, rfcFeedbackDoneParam) {
					done = true
				} else if !namedParamMatcher.MatchString(p) {
					// it's not a named param, we can assume it's a category
					// for more on why this is like this, see frsRequesting
					cats[p] = true
				}
			}
			if id == "" {
				err = rfc.NoRfCIDYetError{}
				return
			}
			return
		}(params)

		if feedbackDone && excludeDone {
			continue
		} else {
			rfcs = append(rfcs, rfc.RfC{ID: rfcID, Categories: categories, FeedbackDone: feedbackDone, PageHolding: title})
		}
	}
	return
}

// extractGANom takes a page name and content that's been nominated for GA,
// and returns the GA nom object.
func extractGANom(content string, title string) (nom ga.Nom) {
	matchedGaTag := gaMatcher.FindStringSubmatch(content)
	// first capture group is name of topic, if applicable
	// second capture group is name of subtopic
	nom = ga.Nom{Topic: matchedGaTag[1], Subtopic: matchedGaTag[2], Article: title}
	return
}
