package domain

type Config struct {
	Version  string `json:"version"`
	Port     int    `json:"port"`
	MockMode bool   `json:"mock_mode"`
}
