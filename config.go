package main

import "time"

type Config struct {
	gwType      string        `toml:"gateway_type"`
	listen_port uint          `toml:"listen_port"`
	deadline    time.Duration `toml:"deadline"`
	remote      Remote
}

type Remote struct {
	scionAddr string `toml:"scion_add"`
}
