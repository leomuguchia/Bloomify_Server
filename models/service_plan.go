package models

type ServicePlan struct {
	ServiceType string   `json:"serviceType"`
	BookingFor  string   `json:"bookingFor"`
	Priority    bool     `json:"priority"`
	Mode        string   `json:"serviceMode"`
	LocationGeo GeoPoint `json:"locationGeo"`
	Date        string   `json:"date"`
	Units       int      `json:"units"`
	UnitType    string   `json:"unitType"`
}
