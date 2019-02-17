package main

import (
	"fmt"
)

func main() {
	fmt.Println("Linker-IM banchmark tools.")
	config := parseConfigure()
	if config == nil {
		return
	}
	if config.PushMode {
		GoPush(config)
	} else {
		GoPull(config)
	}
}
