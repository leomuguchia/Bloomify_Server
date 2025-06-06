package models

type ServicePlan struct {
	ServiceType         string              `json:"serviceType"`
	BookingFor          string              `json:"bookingFor"`
	Priority            bool                `json:"priority"`
	Mode                string              `json:"mode"`
	LocationGeo         GeoPoint            `json:"locationGeo"`
	Date                string              `json:"date"`
	Units               int                 `json:"units"`
	UnitType            string              `json:"unitType"`
	Subscription        bool                `json:"subscription"`
	SubscriptionDetails SubscriptionDetails `json:"subscriptionDetails,omitempty"`
	CustomOption        string              `json:"customOption,omitempty"`
}

const (
	ModeInHome         = "in_home"
	ModeInStore        = "in_store"
	ModePickupDelivery = "pickup_delivery"
	ModeVirtual        = "online"
)

// ServiceMetadata represents the core info for display in listings.
type ServiceMetadata struct {
	ID           string   `json:"id"`
	Icon         string   `json:"icon"`
	UnitType     string   `json:"unitType"`
	ProviderTerm string   `json:"providerTerm"`
	Modes        []string `json:"modes"`
	Category     string   `json:"category"`
}

type ServiceCatalogue struct {
	Service       ServiceMetadata `bson:"service" json:"service" binding:"required"`
	Mode          string          `bson:"mode" json:"mode" binding:"required"`
	CustomOptions []CustomOption  `bson:"customOptions" json:"customOptions" binding:"required"`
	Currency      string          `bson:"currency" json:"currency" binding:"required"`
	ImageURLs     []string        `bson:"imageUrls" json:"imageUrls,omitempty"`
	Price         float64         `bson:"price" json:"price" binding:"required"`
}

type CustomOption struct {
	Option     string  `bson:"option" json:"option" binding:"required"`
	Multiplier float64 `bson:"multiplier" json:"multiplier" binding:"required"`
}

type CustomOptionResponse struct {
	Option string  `json:"option"`
	Price  float64 `json:"price"`
}
