package models

// swagger:model networkPortInfo
type NetworkPortInfo struct {

	// duplex
	// Example: full half
	Duplex string `json:"duplex,omitempty"`

	// 接口数组
	InterfaceNames []string `json:"interfaceNames"`

	// 链路状态
	// Example: 1000 Mbit/s
	LinkSpeed string `json:"linkSpeed,omitempty"`

	// 是否插入网线
	LinkState string `json:"linkState,omitempty"`

	// mac address
	// Example: 02:44:CD:B0:1E:20
	MacAddress string `json:"macAddress,omitempty"`

	// like: the slave of br-lan
	Master string `json:"master,omitempty"`

	// name
	// Example: eth0,eth1
	Name string `json:"name,omitempty"`

	// 接收
	// Example: 2.60 GiB (3715623 pkts.)
	RxPackets string `json:"rx_packets,omitempty"`

	// 发送
	// Example: 2.60 GiB (3715623 pkts.)
	TxPackets string `json:"tx_packets,omitempty"`
}
