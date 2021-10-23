package main

type Config struct {
	gwType      string `toml:"gateway_type"`
	listen_port uint   `toml:"listen_port"`
	remote      Remote
}

type Remote struct {
	scionAddr string `toml:"scion_add"`
}
