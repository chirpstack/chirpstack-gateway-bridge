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
