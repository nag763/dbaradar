package queue

import "github.com/nag763/dbaradar/services/dba-fetcher/internal/fetcher"

func ToSqsEvent(doc fetcher.DbaDoc, eventID, query, chatID string) SqsEvent {
	return SqsEvent{
		EventID:      eventID,
		ID:           doc.ID,
		Heading:      doc.Heading,
		Price:        doc.Price,
		Location:     doc.Location,
		CanonicalURL: doc.CanonicalURL,
		ImageURLs:    doc.ImageURLs,
		Brand:        doc.Brand,
		Timestamp:    doc.Timestamp,
		Extras:       doc.Extras,
		Query:        query,
		ChatID:       chatID,
	}
}
