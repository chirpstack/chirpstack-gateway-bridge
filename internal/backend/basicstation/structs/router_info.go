package structs

// RouterInfoRequest implements the router-info request.
type RouterInfoRequest struct {
	Router EUI64 `json:"router"`
}

// RouterInfoResponse implements the router-info response.
type RouterInfoResponse struct {
	Router EUI64  `json:"router"`
	Muxs   EUI64  `json:"muxs"`
	URI    string `json:"uri"`
	Error  string `json:"error,omitempty"` // only in case of error
}
