package structs

// Version implements the version message.
type Version struct {
	MessageType MessageType `json:"msgtype"`
	Station     string      `json:"station"`
	Firmware    string      `json:"firmware"`
	Package     string      `json:"package"`
	Model       string      `json:"model"`
	Protocol    int         `json:"protocol"`
	//	Features    []string    `json:"features"`
}
