package main

import (
	"os"

	"github.com/justonetree/pan-cli/internal/app"
)

func main() {
	os.Exit(app.Run(app.Options{BinaryName: "pan"}))
}
