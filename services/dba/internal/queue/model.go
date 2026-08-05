package queue

import "github.com/nag763/dbaradar/services/dba-fetcher/internal/fetcher"

type SqsEvent struct {
	EventID      string          `json:"eventId,omitempty"`
	ID           string          `json:"id,omitempty"`
	Heading      string          `json:"heading,omitempty"`
	Price        fetcher.Price   `json:"price,omitempty"`
	Location     string          `json:"location,omitempty"`
	CanonicalURL string          `json:"canonicalUrl,omitempty"`
	ImageURLs    []string        `json:"imageUrls,omitempty"`
	Brand        string          `json:"brand,omitempty"`
	Timestamp    int64           `json:"timestamp,omitempty"`
	Extras       []fetcher.Extra `json:"extras,omitempty"`
	Query        string          `json:"query,omitempty"`
	ChatID       string          `json:"chatId,omitempty"`
}
