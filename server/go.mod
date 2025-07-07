module github.com/outrigdev/outrig/server

go 1.24.3

require (
	github.com/Masterminds/semver/v3 v3.3.1
	github.com/alexflint/go-filemutex v1.3.0
	github.com/emirpasic/gods v1.18.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.3
	github.com/junegunn/fzf v0.62.0
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/outrigdev/outrig v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.9.1
	golang.org/x/sys v0.33.0
)

require (
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/outrigdev/goid v0.2.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	golang.org/x/term v0.29.0 // indirect
)

replace github.com/outrigdev/outrig => ../
