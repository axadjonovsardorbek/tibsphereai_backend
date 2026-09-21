package main

import (
	"tibsphereai/app"
	"tibsphereai/internal/config"
)

func main() {
	cfg := config.Load()

	app.Run(&cfg)
}
