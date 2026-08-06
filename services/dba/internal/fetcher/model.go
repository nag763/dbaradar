package fetcher

type DbaSearchResponse struct {
	Docs []DbaDoc `json:"docs,omitempty"`
}

type DbaDoc struct {
	Type          string   `json:"type,omitempty"`
	ID            string   `json:"id,omitempty"`
	MainSearchKey string   `json:"mainSearchKey,omitempty"`
	Heading       string   `json:"heading,omitempty"`
	Location      string   `json:"location,omitempty"`
	Image         Image    `json:"image,omitempty"`
	Flags         []string `json:"flags,omitempty"`
	Timestamp     int64    `json:"timestamp,omitempty"`
	AdType        int      `json:"adType,omitempty"`
	Labels        []Label  `json:"labels,omitempty"`

	// Handles both canonicalUrl and canonical_url
	CanonicalURL string `json:"canonical_url,omitempty"`

	Extras    []Extra  `json:"extras,omitempty"`
	Price     Price    `json:"price,omitempty"`
	Distance  int      `json:"distance,omitempty"`
	ImageURLs []string `json:"imageUrls,omitempty"`
	Brand     string   `json:"brand,omitempty"`
	AdID      int64    `json:"adId,omitempty"`
}

type Image struct {
	URL  string `json:"url,omitempty"`
	Path string `json:"path,omitempty"`
}

type Label struct {
	ID   string `json:"id,omitempty"`
	Text string `json:"text,omitempty"`
	Type string `json:"type,omitempty"`
}

type Extra struct {
	ID     string   `json:"id,omitempty"`
	Label  string   `json:"label,omitempty"`
	Values []string `json:"values,omitempty"`
}

type Price struct {
	Amount       int    `json:"amount,omitempty"`
	CurrencyCode string `json:"currencyCode,omitempty"`
	PriceUnit    string `json:"priceUnit,omitempty"`
}

func (p Price) DefaultCurrency() string {
	return "DKK"
}

type Metadata struct {
	Source string `json:"source,omitempty"`
}
