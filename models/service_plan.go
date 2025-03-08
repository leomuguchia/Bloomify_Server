package models

type ServicePlan struct {
	Service     string   `json:"service"`
	BookingFor  string   `json:"booking_for"`
	Priority    bool     `json:"priority"`
	Location    string   `json:"location"`
	LocationGeo GeoPoint `json:"location_geo"`
	Date        string   `json:"date"`
	Units       int      `json:"units"`
	UnitType    string   `json:"unitType"`
}
