// Command readme-radar gives you a 30-second "is this repo safe & worth it"
// verdict for any GitHub repository or npm/PyPI package, before you install it.
// See https://github.com/agenticraptor/readme-radar.
package main

import (
	"os"

	"github.com/agenticraptor/readme-radar/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
