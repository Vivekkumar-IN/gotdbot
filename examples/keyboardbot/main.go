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

	client.AddCommandHandler("start", func(c *gotdbot.Client, u *gotdbot.UpdateNewMessage) error {
		msg := u.Message
		userId := msg.SenderID()

		err := msg.Action(c, &gotdbot.SendChatActionOpts{Action: &gotdbot.ChatActionTyping{}})
		if err != nil {
			c.Logger.Error("Failed to send chat action", "err", err)
			return err
		}

		c.Logger.Info("Received /start command", "user_id", userId)

		user, err := c.GetUser(userId)
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

		msg, err = msg.ReplyText(c, text, &gotdbot.SendTextMessageOpts{
			ReplyMarkup: kb,
			ParseMode:   "HTML",
		})
		if err != nil {
			c.Logger.Error("Failed to send welcome message", "err", err)
			return err
		}

		link, err := msg.GetLink(c)
		if err != nil {
			c.Logger.Error("Failed to get message link", "err", err)
		} else {
			c.Logger.Info("Sent welcome message", "link", link.Link)
		}
		return nil
	})

	// /inline - Send message with inline keyboard buttons
	client.AddCommandHandler("inline", func(c *gotdbot.Client, u *gotdbot.UpdateNewMessage) error {
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

		text, err := c.ParseText("This is a Inline keyboard", "Markdown")
		if err != nil {
			c.Logger.Error("Failed to parse text", "err", err)
			return err
		}

		content := &gotdbot.InputMessageText{Text: text}
		opts := &gotdbot.SendMessageOpts{
			ReplyMarkup: kb,
		}

		message, err := c.SendMessage(u.Message.ChatId, content, opts)
		if err != nil {
			c.Logger.Error("Failed to send message", "err", err)
			return err
		}
		c.Logger.Info("Sent message with inline", "message_id", message.Id)

		return nil
	})

	// /keyboard - Send message with reply keyboard
	client.AddCommandHandler("keyboard", func(c *gotdbot.Client, u *gotdbot.UpdateNewMessage) error {
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

		message, err := c.SendMessage(u.Message.ChatId, content, opts)
		if err != nil {
			c.Logger.Error("Failed to send message", "err", err)
			return err
		}
		c.Logger.Info("Sent message with keyboard", "message_id", message.Id)
		return nil
	})

	// /remove - Remove keyboard
	client.AddCommandHandler("remove", func(c *gotdbot.Client, u *gotdbot.UpdateNewMessage) error {
		content := &gotdbot.InputMessageText{
			Text: &gotdbot.FormattedText{
				Text: "Keyboards removed",
			},
		}

		opts := &gotdbot.SendMessageOpts{
			ReplyMarkup: &gotdbot.ReplyMarkupRemoveKeyboard{},
		}

		_, err := c.SendMessage(u.Message.ChatId, content, opts)
		return err
	})

	// /force - Force reply
	client.AddCommandHandler("force", func(c *gotdbot.Client, u *gotdbot.UpdateNewMessage) error {
		content := &gotdbot.InputMessageText{
			Text: &gotdbot.FormattedText{
				Text: "This is a force reply",
			},
		}

		opts := &gotdbot.SendMessageOpts{
			ReplyMarkup: &gotdbot.ReplyMarkupForceReply{},
		}

		_, err := c.SendMessage(u.Message.ChatId, content, opts)

		// _, err = u.Message.ReplyText(c, "This is a force reply", &gotdbot.SendTextMessageOpts{ReplyMarkup: &gotdbot.ReplyMarkupForceReply{}})
		return err
	})

	// CallbackQuery Handler
	client.AddNewCallbackQueryHandler(func(c *gotdbot.Client, u *gotdbot.UpdateNewCallbackQuery) error {
		c.Logger.Info("Received callback query", "message_id", u.MessageId, "chat_id", u.ChatId)
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

			_, err = c.EditMessageText(u.ChatId, inputContent, u.MessageId, &gotdbot.EditMessageTextOpts{
				ReplyMarkup: kb,
			})

			if err != nil {
				c.Logger.Error("Failed to edit message", "error", err)
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
