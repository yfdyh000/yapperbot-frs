package main

import (
	"log"

	"github.com/antonholmquist/jason"
)

// Yes, there is a function to do this already in the library (GetPagesByName).
// No, I don't want to use it. Why? We've already got the page content here -
// making another request to get it again is wasteful when we could just locally
// parse what we already have.
func getContentFromPage(page *jason.Object) (content string, err error) {
	rev, err := page.GetObjectArray("revisions")
	if err != nil {
		log.Println("Failed to get revisions from page, erroring getContentFromPage. Error was ", err)
		return
	}
	content, err = rev[0].GetString("slots", "main", "content")
	if err != nil {
		log.Println("Failed to get main slot content from page, erroring getContentFromPage. Error was", err)
		return
	}
	return
}

func getPagesFromQuery(resp *jason.Object) []*jason.Object {
	query, err := resp.GetObject("query")
	if err != nil {
		switch err.(type) {
		case jason.KeyNotFoundError:
			// no query means no results
			return []*jason.Object{}
		default:
			panic(err)
		}
	}
	pages, err := query.GetObjectArray("pages")
	if err != nil {
		panic(err)
	}
	return pages
}

// Takes a page, and gets the timestamp at which the page was categorised.
// All the errors in this function are Fatal, because frankly,
// if something's gone wrong with the timestamp reading, we're not really
// going to be able to run the algorithm correctly anyway.
func getCategorisationTimestampFromPage(page *jason.Object, category string) (timestamp string) {
	itemCategories, err := page.GetObjectArray("categories")
	if err != nil {
		log.Fatal("Failed to get categories with error message ", err)
	}
	relevantCategory := itemCategories[0]

	timestamp, err = relevantCategory.GetString("timestamp")
	if err != nil {
		log.Fatal("Failed to get categorisation timestamp with error message ", err)
	}
	return
}
