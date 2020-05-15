package main

// frsRequesting is an interface covering all objects that could request FRS.
// At the moment, that's only ga.Nom and rfc.RfC
type frsRequesting interface {
	IncludeHeader(string) bool
	PageTitle() string
	RequestType() string
}
