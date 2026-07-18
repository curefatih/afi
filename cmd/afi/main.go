package main

import (
	"context"
	"log"

	"github.com/curefatih/afi/internal/bootstrap"
)

func main() {
	ctx := context.Background()

	app, err := bootstrap.New(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if err := app.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
