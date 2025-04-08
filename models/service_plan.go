package models

type ServicePlan struct {
	ServiceType string   `json:"serviceType"`
	BookingFor  string   `json:"bookingFor"`
	Priority    bool     `json:"priority"`
	Mode        string   `json:"mode"`
	LocationGeo GeoPoint `json:"locationGeo"`
	Date        string   `json:"date"`
	Units       int      `json:"units"`
	UnitType    string   `json:"unitType"`
}

const (
	ModeInHome         = "in_home"
	ModeInStore        = "in_store"
	ModePickupDelivery = "pickup_delivery"
)

type Service struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Icon         string `json:"icon"`
	UnitType     string `json:"unitType"`
	ProviderTerm string `json:"providerTerm"`
}

// ServiceCatalogue defines the offerings for a service provider.
type ServiceCatalogue struct {
	ServiceType   string             `bson:"serviceType" json:"serviceType,omitempty"`
	Mode          string             `bson:"mode" json:"mode,omitempty"` // e.g., "provider-to-user", "user-to-provider", "pickup/drop-off"
	CustomOptions map[string]float64 `bson:"customOptions" json:"customOptions,omitempty"`
}

// example:
// ServiceCatalogue{
// ServiceType: "cleaning",
// Mode: "provider-to-user",
// CustomOptions: map[string]float64{
// "standard": 1.0,
// "luxury":   1.2,
// "eco":      1.1,
// },
// }
