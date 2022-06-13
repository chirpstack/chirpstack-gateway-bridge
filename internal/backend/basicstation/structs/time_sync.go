package structs

// TimeSyncRequest implements the router-info request.
type TimeSyncRequest struct {
	MessageType MessageType `json:"msgtype"`
	TxTime      int64       `json:"txtime"`
}

// TimeSyncResponse implements the router-info response.
type TimeSyncResponse struct {
	MessageType MessageType `json:"msgtype"`
	TxTime      int64       `json:"txtime"`
	GPSTime     int64       `json:"gpstime"`
}

// TimeSyncGPSTimeTransfer implements the GPS time transfer
// that is initiated by the NS.
type TimeSyncGPSTimeTransfer struct {
	MessageType MessageType `json:"msgtype"`
	XTime       uint64      `json:"xtime"`
	GPSTime     int64       `json:"gpstime"`
}
