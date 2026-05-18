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

	client.AddCommandHandler("start", func(c *gotdbot.Client, u *gotdbot.UpdateNewMessage) error {
		msg := u.Message
		_, err := msg.ReplyText(c, "Welcome! Use /survey to start the survey.\nSend /cancel to stop talking to me", nil)
		return err
	})

	client.AddCommandHandler("survey", func(c *gotdbot.Client, u *gotdbot.UpdateNewMessage) error {
		msg := u.Message
		chatId := msg.ChatId

		timeOut := 30 * time.Second
		stopFilter := gotdbot.FilterText.And(gotdbot.FilterSenderID(msg.SenderID())).And(gotdbot.FilterCommand("cancel"))

		_, err = msg.ReplyText(c, "What is your name?", nil)
		if err != nil {
			return err
		}

		nameMsg, err := c.Ask(chatId, &gotdbot.WaitMessageOpts{Timeout: timeOut, Filter: gotdbot.FilterText.And(gotdbot.FilterSenderID(msg.SenderID())), CancellationFilter: stopFilter})
		if err != nil {
			_, _ = msg.ReplyText(c, err.Error(), nil)
			return nil
		}

		_, err = msg.ReplyText(c, fmt.Sprintf("I see! Please send me a photo of yourself, %s.", nameMsg.Text()), &gotdbot.SendTextMessageOpts{ReplyMarkup: &gotdbot.ReplyMarkupForceReply{InputFieldPlaceholder: "Send a picture"}})
		if err != nil {
			return err
		}

		picMsg, err := c.Ask(chatId, &gotdbot.WaitMessageOpts{Timeout: timeOut, Filter: gotdbot.FilterPhoto.And(gotdbot.FilterSenderID(msg.SenderID())), CancellationFilter: stopFilter})
		if err != nil {
			if errors.Is(err, gotdbot.ConversationCancelled) {
				_, _ = msg.ReplyText(c, "Survey cancelled. Send /survey to start again.", nil)
				return nil
			}

			_, _ = msg.ReplyText(c, "Timeout !", nil)
			return nil
		}

		_, err = msg.ReplyPhoto(c, gotdbot.InputFileRemote{Id: picMsg.RemoteFileID()}, &gotdbot.SendPhotoOpts{Caption: fmt.Sprintf("Nice to meet you, %s!", nameMsg.Text())})
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
