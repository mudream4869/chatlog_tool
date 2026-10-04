//go:build !(js && wasm)

package main

import (
	"embed"
	"encoding/json"
	"flag"
	"io/fs"
	"log"

	"github.com/voilelab/toolgui/toolgui/tgexec"
)

var (
	//go:embed assets
	assets embed.FS
	//go:embed manifest.json
	manifestJSON []byte
	//go:embed head.html
	headHTML string
)

func main() {
	addr := flag.String("addr", "127.0.0.1:3000", "listen address")
	flag.Parse()

	var manifest tgexec.Manifest
	if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
		log.Fatal(err)
	}
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		log.Fatal(err)
	}

	e := tgexec.NewWebExecutor(newApp())
	e.SetManifest(&manifest)
	e.SetAssets(sub)
	e.SetHeadHTML(headHTML)

	log.Printf("Serving on http://%s", *addr)
	if err := e.StartService(*addr); err != nil {
		log.Fatal(err)
	}
}
