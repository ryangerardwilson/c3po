package main

import (
	"os"

	"github.com/ryangerardwilson/c3po/internal/scout"
)

func main() {
	os.Exit(scout.Main(os.Args))
}
