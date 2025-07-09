module github.com/outrigdev/outrig/macosapp

go 1.24.3

require (
	fyne.io/systray v1.11.0
	github.com/Masterminds/semver/v3 v3.3.1
	github.com/outrigdev/outrig v0.0.0-00010101000000-000000000000
	github.com/outrigdev/outrig/server v0.0.0-00010101000000-000000000000
)

require (
	github.com/alexflint/go-filemutex v1.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/outrigdev/goid v0.2.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/term v0.29.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/outrigdev/outrig => ../

replace github.com/outrigdev/outrig/server => ../server
