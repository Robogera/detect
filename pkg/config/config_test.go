package config

import (
	"fmt"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

func TestSanity(t *testing.T) {
	cfg, err := Unmarshal("../../cfg/config.default.toml")
	pretty, err := toml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Can't marshal, err: %s", err)
	}
	fmt.Printf("Config: %s\n", string(pretty))
}

func TestCreate(t *testing.T) {
	err := CreateDefault("../../cfg/empty.toml")
	if err != nil {
		t.Fatalf("Can't create empty config: %s", err)
	}
}
