package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Vivekkumar-IN/gotdbot"
)

func main() {
	apiID := int32(6)
	apiHash := ""

	// Comma separated bot tokens, e.g. "123:ABC,456:DEF"
	tokensStr := ""
	libraryPath := "./libtdjson.so.1.8.64"

	manager := gotdbot.GetDefaultManager(libraryPath)

	// OnNewClient fires once per bot when it is registered.
	// All handlers are attached here so every bot shares the same logic.
	manager.OnNewClient(func(c *gotdbot.Client) {
		c.AddNewMessageHandler(echoHandler, gotdbot.FilterPrivate)
		c.AddCommandHandler("start", startHandler)
	})

	for _, token := range strings.Split(tokensStr, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}

		// Use the numeric bot ID as a unique directory name to avoid
		// TDLib database conflicts between bots running in the same process.
		botID := strings.Split(token, ":")[0]
		dbDir := fmt.Sprintf("tdlib-data-%s", botID)

		config := gotdbot.DefaultClientConfig()
		config.DatabaseDirectory = dbDir
		config.FilesDirectory = dbDir + "/files"

		if _, err := manager.RegisterClient(apiID, apiHash, token, config); err != nil {
			log.Printf("Failed to start bot %s: %v", token, err)
		}
	}

	fmt.Println("Bots are running. Press Ctrl+C to stop.")

	manager.Idle()
}

func echoHandler(client *gotdbot.Client, update *gotdbot.UpdateNewMessage) error {
	m := update.Message

	_, err := m.ReplyText(client, m.Text())
	return err
}

func startHandler(client *gotdbot.Client, update *gotdbot.UpdateNewMessage) error {
	m := update.Message
	_, err := m.ReplyText(client, "Hello! I'm alive and running 🚀")
	return err
}
