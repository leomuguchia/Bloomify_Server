package models

type FeedItem struct {
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	CustomOption string  `json:"customOption"` // E.g. "luxury", "eco-friendly"
	Image        string  `json:"image,omitempty"`
	PriceRange   string  `json:"priceRange,omitempty"`
	ServiceType  string  `json:"serviceType"` // E.g. "cleaning"
	Rating       float64 `json:"rating,omitempty"`
}

type FeedBlock struct {
	Theme       string     `json:"theme"`
	Description string     `json:"description,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
	FeedItems   []FeedItem `json:"feedItems"`
}
