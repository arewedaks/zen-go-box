package config

import (
	"fmt"
	"time"
)

func (c *Config) Validate() error {
	// 1. Core validation
	validCore := false
	for _, b := range c.Core.BinList {
		if c.Core.BinName == b {
			validCore = true
			break
		}
	}
	if !validCore {
		return fmt.Errorf("invalid bin_name: %s, must be one of core.bin_list", c.Core.BinName)
	}

	// 2. Network validation
	switch c.Network.Mode {
	case "tproxy", "redirect", "mixed", "enhance", "tun":
		// valid
	default:
		return fmt.Errorf("invalid network mode: %s, must be one of: tproxy, redirect, mixed, enhance, tun", c.Network.Mode)
	}

	if c.Network.TProxyPort <= 0 || c.Network.TProxyPort > 65535 {
		return fmt.Errorf("invalid tproxy_port: %d", c.Network.TProxyPort)
	}
	if c.Network.RedirPort <= 0 || c.Network.RedirPort > 65535 {
		return fmt.Errorf("invalid redir_port: %d", c.Network.RedirPort)
	}

	// 3. Process duration string parsing
	d, err := time.ParseDuration(c.Process.RestartWindowStr)
	if err != nil {
		return fmt.Errorf("invalid restart_window duration: %w", err)
	}
	c.Process.RestartWindow = d

	d, err = time.ParseDuration(c.Process.RestartBackoffStr)
	if err != nil {
		return fmt.Errorf("invalid restart_backoff duration: %w", err)
	}
	c.Process.RestartBackoff = d

	// 4. Proxy mode validation
	switch c.Proxy.Mode {
	case "blacklist", "whitelist":
		// valid
	default:
		return fmt.Errorf("invalid proxy.mode: %s", c.Proxy.Mode)
	}

	return nil
}
