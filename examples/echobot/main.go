package main

//go:generate go run ../../scripts/tools/get_tdjson.go

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/Vivekkumar-IN/gotdbot"
)

func main() {
	apiID := int32(0)
	apiHash := ""
	botToken := ""

	client, err := gotdbot.NewClient(apiID, apiHash, botToken, &gotdbot.ClientOpts{LibraryPath: "./libtdjson.so.1.8.64"})
	if err != nil {
		panic(err)
	}

	var startTime = time.Now()

	client.AddCommandHandler("start", func(client *gotdbot.Client, msg *gotdbot.Message) error {
		kb := &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{
					{
						Text: "GoTDBot GitHub",
						Type: &gotdbot.InlineKeyboardButtonTypeUrl{
							Url: "https://github.com/Vivekkumar-IN/gotdbot",
						},
					},
				},
			},
		}

		_, err := msg.ReplyText(client, "Hello! I am an echo bot powered by gotdbot "+gotdbot.Version, &gotdbot.SendTextMessageOpts{
			ReplyMarkup: kb,
		})
		if err != nil {
			client.Logger.Errorf("Error sending message: %v", err)
		}
		return nil
	})

	client.AddDeleteMessagesHandler(func(client *gotdbot.Client, u *gotdbot.UpdateDeleteMessages) error {
		client.Logger.Infof("Messages deleted: ids=%v, chat_id=%d", u.MessageIds, u.ChatId)
		return nil
	})

	client.AddCommandHandler("go", func(client *gotdbot.Client, msg *gotdbot.Message) error {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		uptime := time.Since(startTime).Round(time.Second)
		reply := fmt.Sprintf(
			"🟢 Go runtime stats\n\n"+
				"• Goroutines : %d\n"+
				"• CPUs       : %d\n"+
				"• GOMAXPROCS : %d\n"+
				"• Uptime     : %s\n\n"+
				"🧠 Memory\n"+
				"• Alloc      : %.2f MB\n"+
				"• HeapAlloc  : %.2f MB\n"+
				"• Sys        : %.2f MB\n"+
				"• GC cycles  : %d",
			runtime.NumGoroutine(),
			runtime.NumCPU(),
			runtime.GOMAXPROCS(0),
			uptime,
			float64(m.Alloc)/1024/1024,
			float64(m.HeapAlloc)/1024/1024,
			float64(m.Sys)/1024/1024,
			m.NumGC,
		)

		_, err := msg.ReplyText(client, reply)
		return err
	})

	client.OnMessage(func(client *gotdbot.Client, msg *gotdbot.Message) error {
		_, err := msg.Copy(client, msg.ChatId)
		return err
	}, gotdbot.FilterPrivate)

	err = client.Start()
	if err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}

	me, _ := client.GetMe()
	username := ""
	if me.Usernames != nil && len(me.Usernames.ActiveUsernames) > 0 {
		username = me.Usernames.ActiveUsernames[0]
	}
	client.Logger.Infof("Logged in: username=%s, id=%d", username, me.Id)
	client.Idle()
}
