module github.com/kbats183/simple-rtmp-restreamer

go 1.27.0

toolchain go1.27.3

require (
	github.com/go-chi/chi/v5 v5.5.0
	github.com/prometheus/client_golang v1.25.0
	github.com/prometheus/client_model v0.6.3
	github.com/yapingcat/gomedia v0.0.0-20240906162731-17feea57090c
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pandatix/go-abnf v0.5.5 // indirect
	github.com/prometheus/common v0.73.1 // indirect
	github.com/prometheus/procfs v0.23.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/yapingcat/gomedia => github.com/kbats183/gomedia v0.0.0-20250817114334-50ae796beb83
