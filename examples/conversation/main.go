package main

//go:generate go run ../../scripts/tools/get_tdjson.go

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Vivekkumar-IN/gotdbot"
)

func main() {
	apiID := int32(6)
	apiHash := "API_HASH"
	botToken := "BOT_TOKEN"

	client, err := gotdbot.NewClient(apiID, apiHash, botToken, &gotdbot.ClientOpts{LibraryPath: "./libtdjson.so.1.8.64"})
	if err != nil {
		panic(err)
	}

	client.OnCommand("start", func(client *gotdbot.Client, msg *gotdbot.Message) error {
		_, err := msg.ReplyText(client, "Welcome! Use /survey to start the survey.\nSend /cancel to stop talking to me")
		return err
	})

	client.OnCommand("survey", func(client *gotdbot.Client, msg *gotdbot.Message) error {
		chatId := msg.ChatId

		timeOut := 30 * time.Second
		stopFilter := gotdbot.FilterText.And(gotdbot.FilterSenderID(msg.SenderID())).And(gotdbot.FilterCommand("cancel"))

		_, err = msg.ReplyText(client, "What is your name?")
		if err != nil {
			return err
		}

		nameMsg, err := client.Ask(chatId, &gotdbot.WaitMessageOpts{Timeout: timeOut, Filter: gotdbot.FilterText.And(gotdbot.FilterSenderID(msg.SenderID())), CancellationFilter: stopFilter})
		if err != nil {
			_, _ = msg.ReplyText(client, err.Error())
			return nil
		}

		_, err = msg.ReplyText(client, fmt.Sprintf("I see! Please send me a photo of yourself, %s.", nameMsg.Text()), &gotdbot.SendTextMessageOpts{ReplyMarkup: &gotdbot.ReplyMarkupForceReply{InputFieldPlaceholder: "Send a picture"}})
		if err != nil {
			return err
		}

		picMsg, err := client.Ask(chatId, &gotdbot.WaitMessageOpts{Timeout: timeOut, Filter: gotdbot.FilterPhoto.And(gotdbot.FilterSenderID(msg.SenderID())), CancellationFilter: stopFilter})
		if err != nil {
			if errors.Is(err, gotdbot.ConversationCancelled) {
				_, _ = msg.ReplyText(client, "Survey cancelled. Send /survey to start again.")
				return nil
			}

			_, _ = msg.ReplyText(client, "Timeout !")
			return nil
		}

		_, err = msg.ReplyPhoto(client, gotdbot.GetInputFile(picMsg.RemoteFileID()), &gotdbot.SendPhotoOpts{Caption: fmt.Sprintf("Nice to meet you, %s!", nameMsg.Text())})
		if err != nil {
			return err
		}

		return nil
	})

	err = client.Start()
	if err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}
	client.Idle()
}
