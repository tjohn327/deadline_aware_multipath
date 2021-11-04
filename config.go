package main

import "time"

type Config struct {
	GwType      string   `toml:"gateway_type"`
	Listen_port uint     `toml:"listen_port"`
	Deadline    duration `toml:"deadline"`
	FragSize    uint     `toml:"fragment_size"`
	Remote      remote   `toml:"remote"`
}

type remote struct {
	ScionAddr string `toml:"scion_addr"`
}

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}
