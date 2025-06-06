package models

import "time"

type FeedItem struct {
	Title        string               `json:"title"`
	Description  string               `json:"description"`
	CustomOption string               `json:"customOption"` // E.g. "luxury", "eco-friendly"
	Image        string               `json:"image,omitempty"`
	PriceRange   string               `json:"priceRange,omitempty"`
	ServiceType  string               `json:"serviceType"` // E.g. "cleaning"
	Rating       float64              `json:"rating,omitempty"`
	Weight       float64              `json:"weight,omitempty"`
	Providers    []MinimalProviderDTO `json:"providers"`
}

type MinimalProviderDTO struct {
	ID           string   `json:"id"`
	ProviderName string   `json:"providerName"`
	ProfileImage string   `json:"profileImage,omitempty"`
	Location     GeoPoint `json:"location,omitempty"`
	Rating       float64  `json:"rating,omitempty"`
	Verified     bool     `json:"verified,omitempty"`
}

type FeedBlock struct {
	ID          string     `bson:"_id,omitempty" json:"id"`
	Theme       string     `bson:"theme"`
	Description string     `bson:"description,omitempty"`
	Tags        []string   `bson:"tags,omitempty"`
	FeedItems   []FeedItem `bson:"feedItems"`
	CreatedAt   time.Time  `bson:"createdAt"`
	AccessCount int        `bson:"accessCount"`
}

type BlockMeta struct {
	CreatedAt      time.Time `json:"createdAt"`
	LastAccessed   time.Time `json:"lastAccessed"`
	AccessCount    int       `json:"accessCount"`
	Classification string    `json:"classification"` // "trending", "popular", "cold", etc.
}
