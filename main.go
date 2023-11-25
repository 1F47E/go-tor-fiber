package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/cretz/bine/tor"
	"github.com/cretz/bine/torutil/ed25519"
	"github.com/gofiber/fiber/v2"
)

const onionServiceTimeout = 3 * time.Minute
const httpPort = 80

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// start tor
	params := &tor.StartConf{}
	if os.Getenv("DEBUG") == "1" {
		params.DebugWriter = os.Stderr
	}
	torInstance, err := tor.Start(ctx, params)
	if err != nil {
		log.Fatal(err)
	}
	defer torInstance.Close()

	// Wait at most a few minutes to publish the service
	listenCtx, listenCancel := context.WithTimeout(context.Background(), onionServiceTimeout)
	defer listenCancel()

	// Generate a new ed25519 key for the onion service
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	keys, err := ed25519.GenerateKey(rnd)
	if err != nil {
		log.Fatalf("failed to generate onion service key: %v", err)
	}

	// Create onion
	on, err := torInstance.Listen(listenCtx, &tor.ListenConf{
		Version3:    true,
		RemotePorts: []int{httpPort},
		Key:         keys,
	})
	if err != nil {
		log.Fatalf("failed to start onion service: %v", err)
	}
	log.Printf("Generated onion: %s\n", fmt.Sprintf("http://%v.onion", on.ID))

	// Configure Fiber app with custom listener
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello World")
	})
	app.Get("/ping", func(c *fiber.Ctx) error {
		return c.SendString("pong")
	})

	err = app.Listener(on)
	if err != nil {
		log.Fatal(err)
	}
}
