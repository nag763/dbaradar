// Package model defines the data structures shared across the analyzer service.
package model

// DbaEvent represents a listing received from the dba-events SQS queue.
// It is produced by the DBA Ingestor Lambda.
type DbaEvent struct {
	EventID      string   `json:"eventId"`
	Id           string   `json:"id"`
	Heading      string   `json:"heading"`
	Price        Price    `json:"price"`
	Location     string   `json:"location"`
	CanonicalURL string   `json:"canonicalUrl"`
	ImageURLs    []string `json:"imageUrls"`
	Brand        string   `json:"brand"`
	Timestamp    int64    `json:"timestamp"`
	Extras       []Extra  `json:"extras"`
	Query        string   `json:"query"`
}

// Price represents the price of a DBA listing.
type Price struct {
	Amount       int    `json:"amount"`
	CurrencyCode string `json:"currencyCode"`
}

// Extra represents additional metadata on a DBA listing (e.g., condition).
type Extra struct {
	ID     string   `json:"id"`
	Label  string   `json:"label"`
	Values []string `json:"values"`
}

// DealNotification is published to the deal-notifications queue when
// a listing is confirmed as a good deal by the LLM.
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

// LLMVerdict represents the structured response from the Bedrock LLM.
type LLMVerdict struct {
	Verdict string `json:"verdict"`
	Reason  string `json:"reason"`
}

// VerdictGoodDeal is the verdict value indicating the listing is a good deal.
const VerdictGoodDeal = "GOOD_DEAL"
