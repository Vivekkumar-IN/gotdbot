package main

//go:generate go run ../../scripts/tools/get_tdjson.go

import (
	"fmt"
	"log"
	"os"

	"github.com/Vivekkumar-IN/gotdbot"
)

func main() {
	apiID := int32(6)
	apiHash := "API_HASH"
	botToken := "BOT_TOKEN"

	if envID := os.Getenv("API_ID"); envID != "" {
		fmt.Sscanf(envID, "%d", &apiID)
	}
	if envHash := os.Getenv("API_HASH"); envHash != "" {
		apiHash = envHash
	}
	if envToken := os.Getenv("BOT_TOKEN"); envToken != "" {
		botToken = envToken
	}

	client, err := gotdbot.NewClient(apiID, apiHash, botToken, &gotdbot.ClientOpts{LibraryPath: "./libtdjson.so.1.8.64"})
	if err != nil {
		panic(err)
	}

	client.AddCommandHandler("start", func(client *gotdbot.Client, msg *gotdbot.Message) error {
		userId := msg.SenderID()

		err := msg.Action(client, &gotdbot.SendChatActionOpts{Action: &gotdbot.ChatActionTyping{}})
		if err != nil {
			client.Logger.Error("Failed to send chat action", "err", err)
			return err
		}

		client.Logger.Info("Received /start command", "user_id", userId)

		user, err := client.GetUser(userId)
		userName := "User"
		if err == nil {
			userName = user.FirstName
		}

		text := fmt.Sprintf("Hello %s! (gotdbot %s) <tg-emoji emoji-id='5346181118884331907'>🤖</tg-emoji>\nHere are some bot commands:\n\n- /keyboard - show keyboard\n- /inline - show inline keyboard\n- /remove - remove keyboard\n- /force - force reply", userName, gotdbot.Version)

		kb := &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{
					{
						Text: "GitHub",
						Type: &gotdbot.InlineKeyboardButtonTypeUrl{
							Url: "https://github.com/Vivekkumar-IN/gotdbot",
						},
						IconCustomEmojiId: 5271604874419647061,
						Style:             &gotdbot.ButtonStylePrimary{},
					},
				},
			},
		}

		msg, err = msg.ReplyText(client, text, &gotdbot.SendTextMessageOpts{
			ReplyMarkup: kb,
			ParseMode:   "HTML",
		})
		if err != nil {
			client.Logger.Error("Failed to send welcome message", "err", err)
			return err
		}

		link, err := msg.GetLink(client)
		if err != nil {
			client.Logger.Error("Failed to get message link", "err", err)
		} else {
			client.Logger.Info("Sent welcome message", "link", link.Link)
		}
		return nil
	})

	// /inline - Send message with inline keyboard buttons
	client.AddCommandHandler("inline", func(client *gotdbot.Client, msg *gotdbot.Message) error {
		kb := &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{
					{
						Text: "OwO",
						Type: &gotdbot.InlineKeyboardButtonTypeCallback{
							Data: []byte("OwO"),
						},
						Style: &gotdbot.ButtonStylePrimary{},
					},
					{
						Text: "UwU",
						Type: &gotdbot.InlineKeyboardButtonTypeCallback{
							Data: []byte("UwU"),
						},
						Style: &gotdbot.ButtonStyleDanger{},
					},
				},
			},
		}

		text, err := client.ParseText("This is a Inline keyboard", "Markdown")
		if err != nil {
			client.Logger.Error("Failed to parse text", "err", err)
			return err
		}

		content := &gotdbot.InputMessageText{Text: text}
		opts := &gotdbot.SendMessageOpts{
			ReplyMarkup: kb,
		}

		message, err := client.SendMessage(msg.ChatId, content, opts)
		if err != nil {
			client.Logger.Error("Failed to send message", "err", err)
			return err
		}
		client.Logger.Info("Sent message with inline", "message_id", message.Id)

		return nil
	})

	// /keyboard - Send message with reply keyboard
	client.AddCommandHandler("keyboard", func(client *gotdbot.Client, msg *gotdbot.Message) error {
		kb := &gotdbot.ReplyMarkupShowKeyboard{
			Rows: [][]gotdbot.KeyboardButton{
				{
					{
						Text: "OwO",
						Type: &gotdbot.KeyboardButtonTypeText{},
					},
					{
						Text: "UwU",
						Type: &gotdbot.KeyboardButtonTypeText{},
					},
				},
			},
			ResizeKeyboard: true,
			OneTime:        true,
		}

		content := &gotdbot.InputMessageText{
			Text: &gotdbot.FormattedText{
				Text: "This is a keyboard",
			},
		}

		opts := &gotdbot.SendMessageOpts{
			ReplyMarkup: kb,
		}

		message, err := client.SendMessage(msg.ChatId, content, opts)
		if err != nil {
			client.Logger.Error("Failed to send message", "err", err)
			return err
		}
		client.Logger.Info("Sent message with keyboard", "message_id", message.Id)
		return nil
	})

	// /remove - Remove keyboard
	client.AddCommandHandler("remove", func(client *gotdbot.Client, msg *gotdbot.Message) error {
		content := &gotdbot.InputMessageText{
			Text: &gotdbot.FormattedText{
				Text: "Keyboards removed",
			},
		}

		opts := &gotdbot.SendMessageOpts{
			ReplyMarkup: &gotdbot.ReplyMarkupRemoveKeyboard{},
		}

		_, err := client.SendMessage(msg.ChatId, content, opts)
		return err
	})

	// /force - Force reply
	client.AddCommandHandler("force", func(client *gotdbot.Client, msg *gotdbot.Message) error {
		content := &gotdbot.InputMessageText{
			Text: &gotdbot.FormattedText{
				Text: "This is a force reply",
			},
		}

		opts := &gotdbot.SendMessageOpts{
			ReplyMarkup: &gotdbot.ReplyMarkupForceReply{},
		}

		_, err := client.SendMessage(msg.ChatId, content, opts)

		// _, err = msg.ReplyText(client, "This is a force reply", &gotdbot.SendTextMessageOpts{ReplyMarkup: &gotdbot.ReplyMarkupForceReply{}})
		return err
	})

	// CallbackQuery Handler
	client.AddNewCallbackQueryHandler(func(client *gotdbot.Client, u *gotdbot.UpdateNewCallbackQuery) error {
		client.Logger.Info("Received callback query", "message_id", u.MessageId, "chat_id", u.ChatId)
		var data string
		if u.Payload != nil {
			if p, ok := u.Payload.(*gotdbot.CallbackQueryPayloadData); ok {
				data = string(p.Data)
			}
		}

		if data != "" {
			kb := &gotdbot.ReplyMarkupInlineKeyboard{
				Rows: [][]gotdbot.InlineKeyboardButton{
					{
						{
							Text: "GitHub",
							Type: &gotdbot.InlineKeyboardButtonTypeUrl{
								Url: "https://github.com/Vivekkumar-IN/gotdbot",
							},
							IconCustomEmojiId: 5330237710655306682,
							Style:             &gotdbot.ButtonStyleSuccess{},
						},
					},
				},
			}

			inputContent := &gotdbot.InputMessageText{
				Text: &gotdbot.FormattedText{
					Text: fmt.Sprintf("You pressed %s", data),
				},
			}

			_, err = client.EditMessageText(u.ChatId, inputContent, u.MessageId, &gotdbot.EditMessageTextOpts{
				ReplyMarkup: kb,
			})

			if err != nil {
				client.Logger.Error("Failed to edit message", "error", err)
			}
		}

		return nil
	})

	err = client.Start()
	if err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}

	me, _ := client.GetMe()
	if me != nil {
		username := ""
		if me.Usernames != nil && len(me.Usernames.ActiveUsernames) > 0 {
			username = me.Usernames.ActiveUsernames[0]
		}
		client.Logger.Info("Logged in", "username", username, "id", me.Id)
	}

	client.Idle()
}
