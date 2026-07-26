// Drawo is a multiplayer drawing game server.
//
// This is the application entry point. It delegates to the Cobra command tree
// so the same binary can serve the API, run migrations, or run other admin tasks.
package main

import (
	"drawo/cmd"
)

func main() {
	cmd.Execute()
}
