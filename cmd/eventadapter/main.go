package main

import (
	"context"

	"knative.dev/pkg/injection/sharedmain"

)

const (
	ControllerLogKey = "eventAdapterController"
)

func main() {
	ctx := context.Background()

	sharedmain.MainWithContext(ctx, ControllerLogKey, NewController)
}
