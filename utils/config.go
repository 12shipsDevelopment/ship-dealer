package utils

import (
	"log"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Debug    bool
	Client   string
	Token    string
	Node     string
	Database string
	Deal     DealConfig
	Car      CarConfig
	Market   MarketConfig
}

type DealConfig struct {
	Enable       bool
	Miner        string
	Node         string
	Client       string
	Duration     int
	Datacap      int
	DealWait     int  `toml:"deal_wait"`
	PendingLimit int  `toml:"pending_limit"`
	VerifiedDeal bool `toml:"verified_deal"`
}

type CarConfig struct {
	Enable    bool
	ChrunkDir string `toml:"chrunk_dir"`
	CarDir    string `toml:"car_dir"`
	Thread    int
	Size      int
}

type MarketConfig struct {
	Enable     bool
	Miner      string
	RPC        string `toml:"rpc"`
	Token      string
	CarDir     string `toml:"car_dir"`
	ImportWait int    `toml:"import_wait"`
}

func ParseConfig(cfg_path string) (*Config, error) {

	if cfg_path == "" {
		cfg_path = "ship-dealer.toml"
	}

	var cfg *Config

	if _, err := toml.DecodeFile(cfg_path, &cfg); err != nil {
		log.Fatal("Error", err)
	}

	return cfg, nil
}
