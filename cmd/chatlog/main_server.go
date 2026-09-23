//go:build !(js && wasm)

package main

import (
	"flag"
	"log"

	"github.com/voilelab/toolgui/toolgui/tgexec"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:3000", "listen address")
	flag.Parse()

	log.Printf("Serving on http://%s", *addr)
	if err := tgexec.NewWebExecutor(newApp()).StartService(*addr); err != nil {
		log.Fatal(err)
	}
}
