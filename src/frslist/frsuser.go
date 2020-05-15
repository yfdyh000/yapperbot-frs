package frslist

// An FRSUser is a struct representing a user who has signed up for the FRS
type FRSUser struct {
	Username string
	Limit    int16
}

// GetCount takes a header and gets the number of messages sent for that header this month.
func (f FRSUser) GetCount(header string) int16 {
	sentCountMux.Lock()
	defer sentCountMux.Unlock()
	return sentCount[header][f.Username]
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
