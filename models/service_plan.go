package models

type ServicePlan struct {
	ServiceType string   `json:"serviceType"`
	BookingFor  string   `json:"booking_for"`
	Priority    bool     `json:"priority"`
	Mode        string   `json:"serviceMode"`
	LocationGeo GeoPoint `json:"location_geo"`
	Date        string   `json:"date"`
	Units       int      `json:"units"`
	UnitType    string   `json:"unitType"`
}
