package models

type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}
type Attributes struct {
	ActiveGSMOperator     int     `json:"Active GSM Operator"`
	AnalogInput1          float64 `json:"Analog Input 1"`
	BatteryCurrent        float64 `json:"Battery Current"`
	BatteryTemperature    float64 `json:"Battery Temperature"`
	BatteryVoltage        float64 `json:"Battery Voltage"`
	DoorStatus            int     `json:"Door Status"`
	EngineRPM             float64 `json:"Engine RPM"`
	EngineTemperature     float64 `json:"Engine Temperature"`
	EngineWorktime        float64 `json:"Engine Worktime"`
	EngineWorktimeCounted float64 `json:"Engine Worktime (counted)"`
	ExternalVoltage       float64 `json:"External Voltage"`
	FuelConsumed          float64 `json:"Fuel Consumed"`
	FuelConsumedCounted   float64 `json:"Fuel Consumed (counted)"`
	FuelLevel             float64 `json:"Fuel Level"`
	FuelRateGPS           float64 `json:"Fuel Rate GPS"`
	FuelUsedGPS           float64 `json:"Fuel Used GPS"`
	FuelLevel2            float64 `json:"Fuel level"` // Note: Duplicate field name
	GNSSHDOP              float64 `json:"GNSS HDOP"`
	GNSSPDOP              float64 `json:"GNSS PDOP"`
	GNSSStatus            int     `json:"GNSS Status"`
	GSMSignal             int     `json:"GSM Signal"`
	HVBatteryLevel        float64 `json:"HV Battery Level"`
	Ignition              bool    `json:"Ignition"`
	LoadWeight            float64 `json:"Load Weight"`
	Movement              bool    `json:"Movement"`
	OilLevel              bool    `json:"Oil Level"`
	SleepMode             int     `json:"Sleep Mode"`
	TotalMileage          float64 `json:"Total Mileage"`
	TotalMileageCounted   float64 `json:"Total Mileage (counted)"`
	TotalOdometer         float64 `json:"Total Odometer"`
	TripDistance          float64 `json:"Trip Distance"`
	TripOdometer          float64 `json:"Trip Odometer"`
	VehicleSpeed          float64 `json:"Vehicle Speed"`
}
type PositionData struct {
	IMEI           string     `json:"imei"`
	TimestampMs    int64      `json:"timestampMs"`
	Longitude      float64    `json:"lng"`
	Latitude       float64    `json:"lat"`
	Altitude       float64    `json:"altitude"`
	Angle          float64    `json:"angle"`
	EventID        int        `json:"event_id"`
	Speed          float64    `json:"speed"`
	Satellites     int        `json:"satellites"`
	Priority       int        `json:"priority"`
	Ignition       bool       `json:"ignition"`
	GenerationType string     `json:"generationType"`
	Attributes     Attributes `json:"attributes"`
}
