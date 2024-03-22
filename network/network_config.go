package network

import "time"

const defaultResponseHeaderTimeout = 10 * time.Minute

type NetworkConfig struct {
	ResponseHeaderTimeout time.Duration
}

func DefaultNetworkConfig() NetworkConfig {
	return NetworkConfig{
		ResponseHeaderTimeout: defaultResponseHeaderTimeout,
	}
}
