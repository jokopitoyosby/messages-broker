package models

import "time"

type Event struct {
	ID            int                    `json:"id"`
	Attributes    map[string]interface{} `json:"attributes"`
	DeviceID      int                    `json:"deviceId"`
	Type          string                 `json:"type"`
	EventTime     time.Time              `json:"eventTime"`
	PositionID    int                    `json:"positionId"`
	GeofenceID    int                    `json:"geofenceId"`
	MaintenanceID int                    `json:"maintenanceId"`
}

type Device struct {
	ID             int                    `json:"id"`
	Attributes     map[string]interface{} `json:"attributes"`
	GroupID        int                    `json:"groupId"`
	CalendarID     int                    `json:"calendarId"`
	Name           string                 `json:"name"`
	UniqueID       string                 `json:"uniqueId"`
	Status         string                 `json:"status"`
	LastUpdate     time.Time              `json:"lastUpdate"`
	PositionID     int                    `json:"positionId"`
	Phone          *string                `json:"phone"`
	Model          *string                `json:"model"`
	Contact        *string                `json:"contact"`
	Category       *string                `json:"category"`
	Disabled       bool                   `json:"disabled"`
	ExpirationTime *time.Time             `json:"expirationTime"`
}

type DeviceAttributes struct {
	SpeedLimit            float64 `json:"speedLimit"`
	FuelDropThreshold     float64 `json:"fuelDropThreshold"`
	FuelIncreaseThreshold float64 `json:"fuelIncreaseThreshold"`
	DeviceInactivityStart float64 `json:"deviceInactivityStart"`
}

type Data struct {
	Event  Event  `json:"event"`
	Device Device `json:"device"`
}
