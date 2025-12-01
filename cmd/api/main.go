package main

import (
	"context"
	"grpc-user-service/cmd/api/app"
	"grpc-user-service/cmd/api/server"
	"log"
)

// main is the entry point of the application.
func main() {
	if err := run(); err != nil {
		log.Fatalf("application exited with error: %v", err)
	}
}

// run initializes and starts the application server.
func run() error {
	// Create application instance
	application, err := app.New()
	if err != nil {
		return err
	}

	// Setup signal handling
	ctx, cancel := server.WithSignal(context.Background())
	defer cancel()

	// Run application
	return application.Run(ctx)
}
