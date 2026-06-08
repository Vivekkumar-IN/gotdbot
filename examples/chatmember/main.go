package main

//go:generate go run ../../scripts/tools/get_tdjson.go

import (
	"fmt"
	"log"

	"github.com/Vivekkumar-IN/gotdbot"
)

func main() {
	apiID := int32(6)
	apiHash := ""
	botToken := ""

	client, err := gotdbot.NewClient(apiID, apiHash, botToken, &gotdbot.ClientOpts{
		LibraryPath: "./libtdjson.so.1.8.64",
	})

	if err != nil {
		panic(err)
	}

	client.Logger.Info("Starting bot...")
	// Register the UpdateChatMember handler
	// This will trigger on any change in chat/user membership (join, leave, promote, demote, etc.)
	client.AddChatMemberHandler(func(client *gotdbot.Client, u *gotdbot.UpdateChatMember) error {
		chatId := u.ChatId
		me, _ := client.GetMe()

		var userId int64
		var memberName string

		getMemberInfo := func(memberId gotdbot.MessageSender) (int64, string) {
			switch sender := memberId.(type) {
			case *gotdbot.MessageSenderUser:
				return sender.UserId, fmt.Sprintf("User <a href=\"tg://user?id=%d\">%d</a>", sender.UserId, sender.UserId)
			case *gotdbot.MessageSenderChat:
				return sender.ChatId, fmt.Sprintf("Chat %d", sender.ChatId)
			default:
				return 0, "Unknown"
			}
		}

		userId, memberName = getMemberInfo(u.NewChatMember.MemberId)
		oldStatus := getStatusType(u.OldChatMember.Status)
		newStatus := getStatusType(u.NewChatMember.Status)

		var text string

		if oldStatus == "Left" && (newStatus == "Member" || newStatus == "Administrator") {
			// User Joined
			if userId == me.Id {
				text = "Hello! I have joined the chat."
			} else {
				text = fmt.Sprintf("%s joined the chat.", memberName)
			}
		} else if (oldStatus == "Member" || oldStatus == "Administrator") && newStatus == "Left" {
			// User Left or was Kicked
			if userId == me.Id {
				// We left? now we can't send message, but log it.
				client.Logger.Info("Bot left chat", "chat_id", chatId)
				return nil
			}
			text = fmt.Sprintf("%s left or was kicked.", memberName)
		} else if newStatus == "Banned" {
			// User Banned
			// if user blocked the bot, we receive a status changes as "Banned"
			text = fmt.Sprintf("%s was banned.", memberName)
		} else if oldStatus == "Banned" && newStatus == "Left" {
			// User Unbanned
			text = fmt.Sprintf("%s was unbanned.", memberName)
		} else if oldStatus == "Member" && newStatus == "Administrator" {
			// User Promoted
			text = fmt.Sprintf("%s was promoted to Admin.", memberName)
		} else if oldStatus == "Administrator" && newStatus == "Member" {
			// User Demoted
			text = fmt.Sprintf("%s was demoted to Member.", memberName)
		} else {
			// if user unblocks the bot, we receive a status changes as Banned TO-> Member
			client.Logger.Info("Status change", "chat_id", chatId, "user_id", userId, "old", oldStatus, "new", newStatus)
			text = fmt.Sprintf("%s status changed: %s -> %s", memberName, oldStatus, newStatus)
		}

		client.Logger.Info("Chat member update", "chat_id", chatId, "text", text)
		_, err := client.SendTextMessage(chatId, text, &gotdbot.SendTextMessageOpts{
			ParseMode: "HTML",
		})

		if err != nil {
			client.Logger.Error("Failed to send message", "chat_id", chatId, "error", err)
		}

		return nil
	})

	err = client.Start()
	if err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}
	client.Idle()
}

// Helper to get string representation of ChatMemberStatus
func getStatusType(status gotdbot.ChatMemberStatus) string {
	switch status.(type) {
	case *gotdbot.ChatMemberStatusAdministrator:
		return "Administrator"
	case *gotdbot.ChatMemberStatusCreator:
		return "Creator"
	case *gotdbot.ChatMemberStatusMember:
		return "Member"
	case *gotdbot.ChatMemberStatusRestricted:
		return "Restricted"
	case *gotdbot.ChatMemberStatusLeft:
		return "Left"
	case *gotdbot.ChatMemberStatusBanned:
		return "Banned"
	default:
		log.Printf("Unknown ChatMemberStatus type: %T", status)
		return "Unknown"
	}
}
