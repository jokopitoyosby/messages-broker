package models

import "time"

type Position struct {
	ID          int                    `json:"id"`
	Attributes  map[string]interface{} `json:"attributes"`
	DeviceID    int                    `json:"deviceId"`
	Protocol    string                 `json:"protocol"`
	ServerTime  time.Time              `json:"serverTime"`
	DeviceTime  time.Time              `json:"deviceTime"`
	FixTime     time.Time              `json:"fixTime"`
	Outdated    bool                   `json:"outdated"`
	Valid       bool                   `json:"valid"`
	Latitude    float64                `json:"latitude"`
	Longitude   float64                `json:"longitude"`
	Altitude    float64                `json:"altitude"`
	Speed       float64                `json:"speed"`
	Course      float64                `json:"course"`
	Address     *string                `json:"address"`
	Accuracy    float64                `json:"accuracy"`
	Network     *string                `json:"network"`
	GeofenceIDs []int                  `json:"geofenceIds"`
}

type DeviceInfo struct {
	ID             int        `json:"id"`
	Attributes     struct{}   `json:"attributes"`
	GroupID        int        `json:"groupId"`
	CalendarID     int        `json:"calendarId"`
	Name           string     `json:"name"`
	UniqueID       string     `json:"uniqueId"`
	Status         string     `json:"status"`
	LastUpdate     time.Time  `json:"lastUpdate"`
	PositionID     int        `json:"positionId"`
	Phone          *string    `json:"phone"`
	Model          *string    `json:"model"`
	Contact        *string    `json:"contact"`
	Category       *string    `json:"category"`
	Disabled       bool       `json:"disabled"`
	ExpirationTime *time.Time `json:"expirationTime"`
}

type TrackingData struct {
	Position Position   `json:"position"`
	Device   DeviceInfo `json:"device"`
}
