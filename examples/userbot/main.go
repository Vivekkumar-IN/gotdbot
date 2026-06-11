package main

//go:generate go run ../../scripts/tools/get_tdjson.go

import (
	"fmt"
	"log"
	"time"

	"github.com/Vivekkumar-IN/gotdbot"
)

func main() {
	apiID := int32(6)
	apiHash := "API_HASH"

	client, err := gotdbot.NewClient(apiID, apiHash, "", &gotdbot.ClientOpts{
		LibraryPath:           "./libtdjson.so.1.8.64",
		UseFileDatabase:       gotdbot.Bool(true),
		AuthorizationTimeout:  2 * time.Minute,
		QrMode:                true,               // Enable QR code login
		DatabaseEncryptionKey: "my_secret_key_29", // Optional: Set a key to encrypt the local database
	})

	if err != nil {
		panic(err)
	}

	client.OnCommand("hi", func(client *gotdbot.Client, msg *gotdbot.Message) error {
		_, err := msg.ReplyText(client, "Hi, this is from gotdbot!")
		return err
	})

	err = client.Start()
	if err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}

	me, _ := client.GetMe()
	if me != nil {
		fmt.Printf("Current user: %s (ID: %d)\n", me.FirstName, me.Id)
	}

	client.Idle()
}
