package ds

type Config struct {
	// DomainSocketPath is the path to the Unix domain socket. If "" => use default.
	// If "-" => disable domain socket.
	DomainSocketPath string

	// ServerAddr is the TCP address (host:port). If "" => use default.
	// If "-" => disable TCP.
	ServerAddr string
}

type LogLine struct {
	LineNum int64  `json:"linenum"`
	Ts      int64  `json:"ts"`
	Msg     string `json:"msg"`
	Source  string `json:"source,omitempty"`
}
