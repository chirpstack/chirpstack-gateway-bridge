package main

import "github.com/brocaar/lora-gateway-bridge/cmd/lora-gateway-bridge/cmd"

//go:generate ./doc.sh

var version string // set by the compiler

func main() {
	cmd.Execute(version)
}
