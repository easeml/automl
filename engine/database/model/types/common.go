package types

import "time"

// CollectionMetadata contains additional information about a query that returns
// arrays as results. This information aids the caller with navigating the partial results.
type CollectionMetadata struct {
	// The total number of items in a collection that is being accessed. - PROBABLY USELESS
	// TotalCollectionSize int

	// The total size of the result after applying query filters but before pagination.
	TotalResultSize int `json:"total-result-size"`

	// The size of the current page of results that is being returned.
	ReturnedResultSize int `json:"returned-result-size"`

	// The string to pass as a cursor to obtain the next page of results.
	NextPageCursor string `json:"next-page-cursor"`
}

// TimeInterval represents a time interval with specific start and end times.
type TimeInterval struct {
	Start time.Time `bson:"start" json:"start"`
	End   time.Time `bson:"end" json:"end"`
}
