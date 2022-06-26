package utils

import (
	"fmt"
	"testing"
)

func TestConfig(t *testing.T) {
	cfg, _ := ParseConfig("")
	fmt.Println(cfg)
}
