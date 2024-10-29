package option

type TailscaleOutboundOptions struct {
	DialerOptions
	Node string `json:"node,omitempty"`
	ServerOptions
	Network NetworkList `json:"network,omitempty"`
}
