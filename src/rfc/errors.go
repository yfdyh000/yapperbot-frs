package rfc

// NoRfCIDYetError is an error used when an RfC detected does not yet have an ID assigned.
type NoRfCIDYetError struct{}

const noRfCIDYetErrorText string = "An identified RfC does not yet have an assigned RfC ID from Legobot."

func (e NoRfCIDYetError) Error() string {
	return noRfCIDYetErrorText
}
