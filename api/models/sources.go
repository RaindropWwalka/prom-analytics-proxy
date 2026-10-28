package models

// SourcesResponse lists the inventory metadata sources that report at
// least one metric.
type SourcesResponse struct {
	Data []string `json:"data"`
}
