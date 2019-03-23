package config

import (
	"github.com/brocaar/lorawan"
)

// PFConfiguration holds the packet-forwarder configuration.
type PFConfiguration struct {
	MAC            lorawan.EUI64 `mapstructure:"-"`
	MACString      string        `mapstructure:"mac"`
	BaseFile       string        `mapstructure:"base_file"`
	OutputFile     string        `mapstructure:"output_file"`
	RestartCommand string        `mapstructure:"restart_command"`
	Version        string        `mapstructure:"-"`
}
