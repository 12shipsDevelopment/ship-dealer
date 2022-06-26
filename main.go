package main

import (
	"fmt"
	"log"
	"os"

	"github.com/12shipsDevelopment/ship-dealer/dealer"
	"github.com/12shipsDevelopment/ship-dealer/utils"
)

func main() {
	var cfg_path string
	if len(os.Args) > 2 {
		cfg_path = os.Args[1]
	}
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
	cfg, err := utils.ParseConfig(cfg_path)
	if err != nil {
		fmt.Println("invalid config\n", err, cfg_path)
		return
	}
	d := dealer.NewDealer(cfg)
	d.Run()
}
