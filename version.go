package main

// version is set at build time via:
//   go build -ldflags "-X main.version=1.2.3"
// Local builds without ldflags report "dev".
var version = "dev"
