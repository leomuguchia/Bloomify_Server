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
