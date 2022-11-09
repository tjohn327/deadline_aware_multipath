package main

import "time"

func ParseDuration(text string) (time.Duration, error) {
	d, err := time.ParseDuration(text)
	return d, err
}
