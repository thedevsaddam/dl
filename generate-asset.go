// +build generateasset

package main

import (
	"log"
	"net/http"

	"github.com/shurcooL/vfsgen"
)

func main() {
	var fs http.FileSystem = http.Dir("./assets/")
	err := vfsgen.Generate(fs, vfsgen.Options{
		Filename:     "./notifier/asset.go",
		PackageName:  "notifier",
		VariableName: "AssetFS",
	})
	if err != nil {
		log.Fatalln(err)
	}
}
