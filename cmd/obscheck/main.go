package main

import (
	"log"

	tool "github.com/sandeepkv93/everything-backend-starter-kit/internal/tools/obscheck"
)

func main() {
	if err := tool.NewRootCommand().Execute(); err != nil {
		log.Fatal(err)
	}
}
