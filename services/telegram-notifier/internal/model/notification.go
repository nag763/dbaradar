// Package model defines data structures for the telegram notifier service.
package model

// DealNotification represents a confirmed good deal received from the
// deal-notifications SQS queue, produced by the Analyzer service.
type DealNotification struct {
	EventID      string `json:"eventId"`
	Heading      string `json:"heading"`
	Price        Price  `json:"price"`
	CanonicalURL string `json:"canonicalUrl"`
	Location     string `json:"location"`
	Verdict      string `json:"verdict"`
	Reason       string `json:"reason"`
	Query        string `json:"query"`
}

// Price represents the price of a DBA listing.
type Price struct {
	Amount       int    `json:"amount"`
	CurrencyCode string `json:"currencyCode"`
}
