package main

import (
	"context"

	"git.crptech.ru/cloud/cloudgw/internal/app"
)

func main() {
	ctx := context.Background()

	a := app.Init(ctx)

	app.Run(ctx, a)
}
