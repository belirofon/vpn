package model

import "encoding/json"

// ServerStatus represents the current state of the config cache.
type ServerStatus string

const (
	StatusLoading ServerStatus = "loading"
	StatusTesting ServerStatus = "testing"
	StatusReady   ServerStatus = "ready"
	StatusError   ServerStatus = "error"
)

// VpnConfig represents a parsed and tested VPN configuration.
type VpnConfig struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Server    string `json:"server"`
	Port      int    `json:"port"`
	Protocol  string `json:"protocol"`
	UUID      string `json:"uuid,omitempty"`
	Password  string `json:"password,omitempty"`
	TLS       string `json:"tls,omitempty"`
	Network   string `json:"network,omitempty"`
	Host      string `json:"host,omitempty"`
	Path      string `json:"path,omitempty"`
	SNI       string `json:"sni,omitempty"`
	FP        string `json:"fp,omitempty"`
	ALPN      string `json:"alpn,omitempty"`
	Pbk       string `json:"pbk,omitempty"`
	Sid       string `json:"sid,omitempty"`
	LatencyMs     int64            `json:"latency_ms"`
	Country       string           `json:"country"`
	RawLink       string           `json:"raw_link,omitempty"`
	SingboxConfig *json.RawMessage `json:"singbox_config,omitempty"`
}

type ConfigListResponse struct {
	Configs []VpnConfig `json:"configs"`
	Updated string      `json:"updated"`
	Total   int         `json:"total"`
}

type BestConfigResponse struct {
	Config  *VpnConfig `json:"config"`
	Updated string     `json:"updated"`
}

type StatusResponse struct {
	Status        ServerStatus `json:"status"`
	Message       string       `json:"message,omitempty"`
	ConfigsTested int          `json:"configs_tested,omitempty"`
	BestName      string       `json:"best_name,omitempty"`
	BestLatency   int64        `json:"best_latency_ms,omitempty"`
	Updated       string       `json:"updated,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// WarpConfig represents a Cloudflare WARP WireGuard configuration.
type WarpConfig struct {
	Protocol        string `json:"protocol"`
	PrivateKey      string `json:"private_key"`
	AddressV4       string `json:"address_v4"`
	AddressV6       string `json:"address_v6"`
	DNSServers      string `json:"dns"`
	ServerPublicKey string `json:"server_public_key"`
	Endpoint        string `json:"endpoint"`
	ClientID        string `json:"client_id,omitempty"`
	LatencyMs       int64  `json:"latency_ms"`
}

type WarpConfigResponse struct {
	Config  *WarpConfig `json:"config"`
	Updated string      `json:"updated"`
}

type BestConfigListResponse struct {
	Configs []VpnConfig `json:"configs"`
	Total   int         `json:"total"`
}

const ErrNoAvailableConfigs = "no_available_configs"
