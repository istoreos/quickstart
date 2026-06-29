package models

type DHNSDhcpValidRequest struct {
	Ip      string `json:"ip"`
	Gateway string `json:"gateway"`
	Subnet  string `json:"subnet"`
	Dns     string `json:"dns"`
}
