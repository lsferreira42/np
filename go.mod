module github.com/lsferreira42/np

go 1.20

require (
	github.com/grandcat/zeroconf v1.0.0
	github.com/klauspost/compress v1.17.2
)

// Forcing more recent versions of dependencies
replace (
	github.com/grandcat/zeroconf => github.com/grandcat/zeroconf v1.0.0
	golang.org/x/net => golang.org/x/net v0.10.0
)

require (
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/miekg/dns v1.1.27 // indirect
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519 // indirect
	golang.org/x/net v0.6.0 // indirect
	golang.org/x/sys v0.8.0 // indirect
)
