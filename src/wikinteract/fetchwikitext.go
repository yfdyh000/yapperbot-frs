package wikinteract

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
