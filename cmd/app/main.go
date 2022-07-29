package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"parabellum.crawler/internal/network"
)

func main() {
	app := new(Config)

	app.initPubSub()
	defer app.closePubSub()

	exitCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app.ClientGrpc = network.NewClient(os.Getenv("GRPC_ADDR"))
	defer app.ClientGrpc.Close()

	for {
		err := app.ExecuteNextTask(exitCtx)
		if err != nil {
			continue
		}

		select {
		case <-exitCtx.Done():
			log.Println("Exiting on termination signal")

			return
		default:
		}
	}
}
