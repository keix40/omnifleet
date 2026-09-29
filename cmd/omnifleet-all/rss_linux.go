//go:build linux

package main

import (
	"log"
	"os"
	"strings"
)

func logProcessRSS() {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			log.Printf("omnifleet-all %s", strings.TrimSpace(line))
			return
		}
	}
}
