package main

import (
	"os"

	"github.com/onetreegrove/panctl/internal/app"
)

func main() {
	os.Exit(app.Run(app.Options{BinaryName: "panctl"}))
}
