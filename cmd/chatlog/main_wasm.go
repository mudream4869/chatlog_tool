//go:build js && wasm

package main

import "github.com/voilelab/toolgui/toolgui/tgwasm"

func main() {
	app := newApp()
	// Static hosts can't route paths, so pages live in the hash.
	app.SetHashPageNameMode(true)
	tgwasm.NewExecutor(app).Run()
}
