package service

type GuideDDNSInterfaceSnapshot struct {
	InterfaceName string
	DeviceName    string
	IP            string
	Gateway       string
	Proto         string
	DNS           []string
}

type GuideDDNSOutboundSnapshot struct {
	IPv4 *GuideDDNSInterfaceSnapshot
	IPv6 *GuideDDNSInterfaceSnapshot
}

type GuideDdnstoConfigSnapshot struct {
	Enabled bool
	Token   string
	Address string
}
