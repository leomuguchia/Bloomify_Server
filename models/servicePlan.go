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
	ModeOnline         = "online"
)

type CapacityMode string

const (
	CapacitySingleUse CapacityMode = "single"    // One booking per slot (e.g., 1-on-1)
	CapacityByUnit    CapacityMode = "by_unit"   // Capacity in measurable units (kg, pets, kids, people)
	CapacityByWorker  CapacityMode = "by_worker" // Capacity = number of workers × duration (in hours)
)

type Service struct {
	ID           string       `json:"id"`
	Icon         string       `json:"icon"`
	UnitType     string       `json:"unitType"`
	ProviderTerm string       `json:"providerTerm"`
	Modes        []string     `json:"modes"`
	CapacityMode CapacityMode `json:"capacityMode"`
}

type ServiceCatalogue struct {
	Service       Service        `bson:"service" json:"service"`
	Mode          string         `bson:"mode" json:"mode"`
	CustomOptions []CustomOption `bson:"customOptions" json:"customOptions"`
	Currency      string         `bson:"currency" json:"currency"`
	ImageURLs     []string       `bson:"imageUrls" json:"imageUrls"`
}

type CustomOption struct {
	Option     string  `bson:"option" json:"option"`
	Multiplier float64 `bson:"multiplier" json:"multiplier"`
}

type CustomOptionResponse struct {
	Option string  `json:"option"`
	Price  float64 `json:"price"`
}
