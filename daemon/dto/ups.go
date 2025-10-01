package dto

import "time"

// UPSStatus contains UPS status information
type UPSStatus struct {
	Connected     bool      `json:"connected"`
	Status        string    `json:"status"`
	LoadPercent   float64   `json:"load_percent"`
	BatteryCharge float64   `json:"battery_charge_percent"`
	RuntimeLeft   int       `json:"runtime_left_seconds"`
	PowerWatts    float64   `json:"power_watts"`
	NominalPower  float64   `json:"nominal_power_watts"`
	Model         string    `json:"model"`
	Timestamp     time.Time `json:"timestamp"`
}
