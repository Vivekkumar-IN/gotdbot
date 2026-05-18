package gotdbot

import (
	"fmt"
	"html"
	"os"
	"strings"
)

func GetFormattedText(c *Client, text string, entities []TextEntity, parseMode string) (*FormattedText, error) {
	if len(entities) > 0 {
		return &FormattedText{
			Text:     text,
			Entities: entities,
		}, nil
	} else if parseMode != "" {
		ft, err := c.ParseText(text, parseMode)
		if err == nil {
			return ft, nil
		}
		return nil, err
	}
	return &FormattedText{Text: text}, nil
}

func GetInputFile(path string) InputFile {
	if _, err := os.Stat(path); err == nil {
		return InputFileLocal{Path: path}
	}

	return InputFileRemote{Id: path}
}

// EscapeHTML escapes HTML characters in the given text.
func EscapeHTML(text string) string {
	return html.EscapeString(text)
}

// EscapeMarkdown escapes Markdown characters in the given text.
func EscapeMarkdown(text string, version int) string {
	var chars string
	if version == 1 {
		chars = "_*`[\\"
	} else {
		chars = "_*[]()~`>#+-=|{}.!\\"
	}
	var b strings.Builder
	for _, c := range text {
		if strings.ContainsRune(chars, c) {
			b.WriteRune('\\')
		}
		b.WriteRune(c)
	}
	return b.String()
}

// Mention returns a text mention for the given user ID.
func Mention(text string, userId int64, isHtml bool, escape bool) string {
	if escape {
		if isHtml {
			text = EscapeHTML(text)
		} else {
			text = EscapeMarkdown(text, 2)
		}
	}
	if isHtml {
		return fmt.Sprintf("<a href=\"tg://user?id=%d\">%s</a>", userId, text)
	}
	return fmt.Sprintf("[%s](tg://user?id=%d)", text, userId)
}

// sendMessageWithContent is a helper function to send a message with content
func (c *Client) sendMessageWithContent(
	chatId int64,
	content InputMessageContent,
	options *MessageSendOptions,
	topicId MessageTopic,
	quote *InputTextQuote,
	replyTo InputMessageReplyTo,
	replyToMessageId int64,
	replyMarkup ReplyMarkup,
) (*Message, error) {
	if replyToMessageId > 0 {
		replyTo = &InputMessageReplyToMessage{
			MessageId: replyToMessageId,
			Quote:     quote,
		}
	}

	if c.config.LoadMessagesBeforeReply && replyToMessageId > 0 {
		_, _ = c.GetMessage(chatId, replyToMessageId)
	}

	return c.SendMessage(chatId, content, &SendMessageOpts{
		TopicId:     topicId,
		ReplyTo:     replyTo,
		Options:     options,
		ReplyMarkup: replyMarkup,
	})
}

// SendTextMessageOpts contains optional parameters for SendTextMessage
type SendTextMessageOpts struct {
	ParseMode             string
	Entities              []TextEntity
	DisableWebPagePreview bool
	Url                   string
	ForceSmallMedia       bool
	ForceLargeMedia       bool
	ShowAboveText         bool
	DisableNotification   bool
	ProtectContent        bool
	AllowPaidBroadcast    bool
	TopicId               MessageTopic
	Quote                 *InputTextQuote
	ReplyTo               InputMessageReplyTo
	ReplyToMessageID      int64
	ReplyMarkup           ReplyMarkup
	ClearDraft            bool
	EffectId              int64
}

// SendTextMessage sends a text message to chat
func (c *Client) SendTextMessage(chatId int64, text string, opts ...*SendTextMessageOpts) (*Message, error) {
	opt := getVariadic(opts, &SendTextMessageOpts{})

	formattedText, err := GetFormattedText(c, text, opt.Entities, opt.ParseMode)
	if err != nil {
		return nil, err
	}

	linkPreviewOptions := &LinkPreviewOptions{
		IsDisabled:      opt.DisableWebPagePreview,
		Url:             opt.Url,
		ForceSmallMedia: opt.ForceSmallMedia,
		ForceLargeMedia: opt.ForceLargeMedia,
		ShowAboveText:   opt.ShowAboveText,
	}

	content := &InputMessageText{
		Text:               formattedText,
		LinkPreviewOptions: linkPreviewOptions,
		ClearDraft:         opt.ClearDraft,
	}

	return c.sendMessageWithContent(chatId, content, &MessageSendOptions{
		DisableNotification: opt.DisableNotification,
		ProtectContent:      opt.ProtectContent,
		AllowPaidBroadcast:  opt.AllowPaidBroadcast,
		EffectId:            opt.EffectId,
	}, opt.TopicId, opt.Quote, opt.ReplyTo, opt.ReplyToMessageID, opt.ReplyMarkup)
}

// SendPhotoOpts contains optional parameters for SendPhoto
type SendPhotoOpts struct {
	Caption             string
	CaptionEntities     []TextEntity
	ParseMode           string
	AddedStickerFileIds []int32
	Width               int32
	Height              int32
	SelfDestructType    MessageSelfDestructType
	DisableNotification bool
	ProtectContent      bool
	AllowPaidBroadcast  bool
	HasSpoiler          bool
	TopicId             MessageTopic
	Quote               *InputTextQuote
	ReplyTo             InputMessageReplyTo
	ReplyToMessageID    int64
	ReplyMarkup         ReplyMarkup
	Thumbnail           *InputThumbnail
	EffectId            int64
}

// SendPhoto sends a photo to chat
func (c *Client) SendPhoto(chatId int64, photo InputFile, opts ...*SendPhotoOpts) (*Message, error) {
	opt := getVariadic(opts, &SendPhotoOpts{})

	caption, err := GetFormattedText(c, opt.Caption, opt.CaptionEntities, opt.ParseMode)
	if err != nil {
		return nil, err
	}

	content := &InputMessagePhoto{
		Photo:               photo,
		Thumbnail:           opt.Thumbnail,
		AddedStickerFileIds: opt.AddedStickerFileIds,
		Width:               opt.Width,
		Height:              opt.Height,
		Caption:             caption,
		SelfDestructType:    opt.SelfDestructType,
		HasSpoiler:          opt.HasSpoiler,
	}

	return c.sendMessageWithContent(chatId, content, &MessageSendOptions{
		DisableNotification: opt.DisableNotification,
		ProtectContent:      opt.ProtectContent,
		AllowPaidBroadcast:  opt.AllowPaidBroadcast,
		EffectId:            opt.EffectId,
	}, opt.TopicId, opt.Quote, opt.ReplyTo, opt.ReplyToMessageID, opt.ReplyMarkup)
}

// SendVideoOpts contains optional parameters for SendVideo
type SendVideoOpts struct {
	Caption             string
	CaptionEntities     []TextEntity
	ParseMode           string
	AddedStickerFileIds []int32
	SupportsStreaming   bool
	Duration            int32
	Width               int32
	Height              int32
	SelfDestructType    MessageSelfDestructType
	DisableNotification bool
	ProtectContent      bool
	AllowPaidBroadcast  bool
	HasSpoiler          bool
	TopicId             MessageTopic
	Quote               *InputTextQuote
	ReplyTo             InputMessageReplyTo
	ReplyToMessageID    int64
	ReplyMarkup         ReplyMarkup
	Thumbnail           *InputThumbnail
	EffectId            int64
}

// SendVideo sends a video to chat
func (c *Client) SendVideo(chatId int64, video InputFile, opts ...*SendVideoOpts) (*Message, error) {
	opt := getVariadic(opts, &SendVideoOpts{})

	caption, err := GetFormattedText(c, opt.Caption, opt.CaptionEntities, opt.ParseMode)
	if err != nil {
		return nil, err
	}

	content := &InputMessageVideo{
		Video:               video,
		Thumbnail:           opt.Thumbnail,
		AddedStickerFileIds: opt.AddedStickerFileIds,
		Duration:            opt.Duration,
		Width:               opt.Width,
		Height:              opt.Height,
		SupportsStreaming:   opt.SupportsStreaming,
		Caption:             caption,
		SelfDestructType:    opt.SelfDestructType,
		HasSpoiler:          opt.HasSpoiler,
	}

	return c.sendMessageWithContent(chatId, content, &MessageSendOptions{
		DisableNotification: opt.DisableNotification,
		ProtectContent:      opt.ProtectContent,
		AllowPaidBroadcast:  opt.AllowPaidBroadcast,
		EffectId:            opt.EffectId,
	}, opt.TopicId, opt.Quote, opt.ReplyTo, opt.ReplyToMessageID, opt.ReplyMarkup)
}

// SendAnimationOpts contains optional parameters for SendAnimation
type SendAnimationOpts struct {
	Caption             string
	CaptionEntities     []TextEntity
	ParseMode           string
	AddedStickerFileIds []int32
	Duration            int32
	Width               int32
	Height              int32
	DisableNotification bool
	ProtectContent      bool
	AllowPaidBroadcast  bool
	HasSpoiler          bool
	TopicId             MessageTopic
	Quote               *InputTextQuote
	ReplyTo             InputMessageReplyTo
	ReplyToMessageID    int64
	ReplyMarkup         ReplyMarkup
	Thumbnail           *InputThumbnail
	EffectId            int64
}

// SendAnimation sends an animation to chat
func (c *Client) SendAnimation(chatId int64, animation InputFile, opts ...*SendAnimationOpts) (*Message, error) {
	opt := getVariadic(opts, &SendAnimationOpts{})

	caption, err := GetFormattedText(c, opt.Caption, opt.CaptionEntities, opt.ParseMode)
	if err != nil {
		return nil, err
	}

	content := &InputMessageAnimation{
		Animation:           animation,
		Thumbnail:           opt.Thumbnail,
		AddedStickerFileIds: opt.AddedStickerFileIds,
		Duration:            opt.Duration,
		Width:               opt.Width,
		Height:              opt.Height,
		Caption:             caption,
		HasSpoiler:          opt.HasSpoiler,
	}

	return c.sendMessageWithContent(chatId, content, &MessageSendOptions{
		DisableNotification: opt.DisableNotification,
		ProtectContent:      opt.ProtectContent,
		AllowPaidBroadcast:  opt.AllowPaidBroadcast,
		EffectId:            opt.EffectId,
	}, opt.TopicId, opt.Quote, opt.ReplyTo, opt.ReplyToMessageID, opt.ReplyMarkup)
}

// SendAudioOpts contains optional parameters for SendAudio
type SendAudioOpts struct {
	Caption             string
	CaptionEntities     []TextEntity
	ParseMode           string
	Title               string
	Performer           string
	Duration            int32
	DisableNotification bool
	ProtectContent      bool
	AllowPaidBroadcast  bool
	TopicId             MessageTopic
	Quote               *InputTextQuote
	ReplyTo             InputMessageReplyTo
	ReplyToMessageID    int64
	ReplyMarkup         ReplyMarkup
	AlbumCoverThumbnail *InputThumbnail
	EffectId            int64
}

// SendAudio sends an audio to chat
func (c *Client) SendAudio(chatId int64, audio InputFile, opts ...*SendAudioOpts) (*Message, error) {
	opt := getVariadic(opts, &SendAudioOpts{})

	caption, err := GetFormattedText(c, opt.Caption, opt.CaptionEntities, opt.ParseMode)
	if err != nil {
		return nil, err
	}

	content := &InputMessageAudio{
		Audio:               audio,
		AlbumCoverThumbnail: opt.AlbumCoverThumbnail,
		Title:               opt.Title,
		Performer:           opt.Performer,
		Duration:            opt.Duration,
		Caption:             caption,
	}

	return c.sendMessageWithContent(chatId, content, &MessageSendOptions{
		DisableNotification: opt.DisableNotification,
		ProtectContent:      opt.ProtectContent,
		AllowPaidBroadcast:  opt.AllowPaidBroadcast,
		EffectId:            opt.EffectId,
	}, opt.TopicId, opt.Quote, opt.ReplyTo, opt.ReplyToMessageID, opt.ReplyMarkup)
}

// SendDocumentOpts contains optional parameters for SendDocument
type SendDocumentOpts struct {
	Caption                     string
	CaptionEntities             []TextEntity
	ParseMode                   string
	DisableContentTypeDetection bool
	DisableNotification         bool
	ProtectContent              bool
	AllowPaidBroadcast          bool
	TopicId                     MessageTopic
	Quote                       *InputTextQuote
	ReplyTo                     InputMessageReplyTo
	ReplyToMessageID            int64
	ReplyMarkup                 ReplyMarkup
	Thumbnail                   *InputThumbnail
	EffectId                    int64
}

// SendDocument sends a document to chat
func (c *Client) SendDocument(chatId int64, document InputFile, opts ...*SendDocumentOpts) (*Message, error) {
	opt := getVariadic(opts, &SendDocumentOpts{})

	caption, err := GetFormattedText(c, opt.Caption, opt.CaptionEntities, opt.ParseMode)
	if err != nil {
		return nil, err
	}

	content := &InputMessageDocument{
		Document:                    document,
		Thumbnail:                   opt.Thumbnail,
		DisableContentTypeDetection: opt.DisableContentTypeDetection,
		Caption:                     caption,
	}

	return c.sendMessageWithContent(chatId, content, &MessageSendOptions{
		DisableNotification: opt.DisableNotification,
		ProtectContent:      opt.ProtectContent,
		AllowPaidBroadcast:  opt.AllowPaidBroadcast,
		EffectId:            opt.EffectId,
	}, opt.TopicId, opt.Quote, opt.ReplyTo, opt.ReplyToMessageID, opt.ReplyMarkup)
}

// SendVoiceOpts contains optional parameters for SendVoice
type SendVoiceOpts struct {
	Caption             string
	CaptionEntities     []TextEntity
	ParseMode           string
	Duration            int32
	Waveform            []byte
	DisableNotification bool
	ProtectContent      bool
	AllowPaidBroadcast  bool
	TopicId             MessageTopic
	Quote               *InputTextQuote
	ReplyTo             InputMessageReplyTo
	ReplyToMessageID    int64
	ReplyMarkup         ReplyMarkup
	EffectId            int64
}

// SendVoice sends a voice note to chat
func (c *Client) SendVoice(chatId int64, voice InputFile, opts ...*SendVoiceOpts) (*Message, error) {
	opt := getVariadic(opts, &SendVoiceOpts{})

	caption, err := GetFormattedText(c, opt.Caption, opt.CaptionEntities, opt.ParseMode)
	if err != nil {
		return nil, err
	}

	content := &InputMessageVoiceNote{
		VoiceNote: voice,
		Waveform:  opt.Waveform,
		Duration:  opt.Duration,
		Caption:   caption,
	}

	return c.sendMessageWithContent(chatId, content, &MessageSendOptions{
		DisableNotification: opt.DisableNotification,
		ProtectContent:      opt.ProtectContent,
		AllowPaidBroadcast:  opt.AllowPaidBroadcast,
		EffectId:            opt.EffectId,
	}, opt.TopicId, opt.Quote, opt.ReplyTo, opt.ReplyToMessageID, opt.ReplyMarkup)
}

// SendVideoNoteOpts contains optional parameters for SendVideoNote
type SendVideoNoteOpts struct {
	Duration            int32
	Length              int32
	DisableNotification bool
	ProtectContent      bool
	AllowPaidBroadcast  bool
	TopicId             MessageTopic
	Quote               *InputTextQuote
	ReplyTo             InputMessageReplyTo
	ReplyToMessageID    int64
	ReplyMarkup         ReplyMarkup
	Thumbnail           *InputThumbnail
	EffectId            int64
}

// SendVideoNote sends a video note to chat
func (c *Client) SendVideoNote(chatId int64, videoNote InputFile, opts ...*SendVideoNoteOpts) (*Message, error) {
	opt := getVariadic(opts, &SendVideoNoteOpts{})

	content := &InputMessageVideoNote{
		VideoNote: videoNote,
		Thumbnail: opt.Thumbnail,
		Duration:  opt.Duration,
		Length:    opt.Length,
	}

	return c.sendMessageWithContent(chatId, content, &MessageSendOptions{
		DisableNotification: opt.DisableNotification,
		ProtectContent:      opt.ProtectContent,
		AllowPaidBroadcast:  opt.AllowPaidBroadcast,
		EffectId:            opt.EffectId,
	}, opt.TopicId, opt.Quote, opt.ReplyTo, opt.ReplyToMessageID, opt.ReplyMarkup)
}

// SendStickerOpts contains optional parameters for SendSticker
type SendStickerOpts struct {
	Emoji               string
	Width               int32
	Height              int32
	DisableNotification bool
	ProtectContent      bool
	AllowPaidBroadcast  bool
	TopicId             MessageTopic
	Quote               *InputTextQuote
	ReplyTo             InputMessageReplyTo
	ReplyToMessageID    int64
	ReplyMarkup         ReplyMarkup
	Thumbnail           *InputThumbnail
	EffectId            int64
}

// SendSticker sends a sticker to chat
func (c *Client) SendSticker(chatId int64, sticker InputFile, opts ...*SendStickerOpts) (*Message, error) {
	opt := getVariadic(opts, &SendStickerOpts{})

	content := &InputMessageSticker{
		Sticker:   sticker,
		Thumbnail: opt.Thumbnail,
		Width:     opt.Width,
		Height:    opt.Height,
		Emoji:     opt.Emoji,
	}

	return c.sendMessageWithContent(chatId, content, &MessageSendOptions{
		DisableNotification: opt.DisableNotification,
		ProtectContent:      opt.ProtectContent,
		AllowPaidBroadcast:  opt.AllowPaidBroadcast,
		EffectId:            opt.EffectId,
	}, opt.TopicId, opt.Quote, opt.ReplyTo, opt.ReplyToMessageID, opt.ReplyMarkup)
}

// SendCopyOpts contains optional parameters for SendCopy
type SendCopyOpts struct {
	InGameShare         bool
	ReplaceCaption      bool
	NewCaption          string
	NewCaptionEntities  []TextEntity
	ParseMode           string
	DisableNotification bool
	ProtectContent      bool
	AllowPaidBroadcast  bool
	TopicId             MessageTopic
	Quote               *InputTextQuote
	ReplyTo             InputMessageReplyTo
	ReplyMarkup         ReplyMarkup
	ReplyToMessageID    int64
	EffectId            int64
}

// SendCopy copies a message to chat
func (c *Client) SendCopy(chatId int64, fromChatId int64, messageId int64, opts ...*SendCopyOpts) (*Message, error) {
	opt := getVariadic(opts, &SendCopyOpts{})

	caption, err := GetFormattedText(c, opt.NewCaption, opt.NewCaptionEntities, opt.ParseMode)
	if err != nil {
		return nil, err
	}
	content := &InputMessageForwarded{
		FromChatId:  fromChatId,
		MessageId:   messageId,
		InGameShare: opt.InGameShare,
		CopyOptions: &MessageCopyOptions{
			SendCopy:       true,
			ReplaceCaption: opt.ReplaceCaption,
			NewCaption:     caption,
		},
	}

	return c.sendMessageWithContent(chatId, content, &MessageSendOptions{
		DisableNotification: opt.DisableNotification,
		ProtectContent:      opt.ProtectContent,
		AllowPaidBroadcast:  opt.AllowPaidBroadcast,
		EffectId:            opt.EffectId,
	}, opt.TopicId, opt.Quote, opt.ReplyTo, opt.ReplyToMessageID, opt.ReplyMarkup)
}

// ForwardMessageOpts contains optional parameters for ForwardMessage
type ForwardMessageOpts struct {
	InGameShare         bool
	DisableNotification bool
	EffectId            int64
}

// ForwardMessage forwards a message to chat
func (c *Client) ForwardMessage(chatId int64, fromChatId int64, messageId int64, opts ...*ForwardMessageOpts) (*Message, error) {
	opt := getVariadic(opts, &ForwardMessageOpts{})

	content := &InputMessageForwarded{
		FromChatId:  fromChatId,
		MessageId:   messageId,
		InGameShare: opt.InGameShare,
	}

	return c.sendMessageWithContent(chatId, content, &MessageSendOptions{
		DisableNotification: opt.DisableNotification,
		EffectId:            opt.EffectId,
	}, nil, nil, nil, 0, nil)
}

// EditTextMessageOpts contains optional parameters for EditTextMessage
type EditTextMessageOpts struct {
	ParseMode             string
	Entities              []TextEntity
	DisableWebPagePreview bool
	Url                   string
	ForceSmallMedia       bool
	ForceLargeMedia       bool
	ShowAboveText         bool
	ReplyMarkup           ReplyMarkup
}

// EditTextMessage edits a text message
func (c *Client) EditTextMessage(chatId int64, messageId int64, text string, opts ...*EditTextMessageOpts) (*Message, error) {
	opt := getVariadic(opts, &EditTextMessageOpts{})

	if !*c.config.UseMessageDatabase {
		if _, err := c.GetMessage(chatId, messageId); err != nil {
			return nil, err
		}
	}

	formattedText, err := GetFormattedText(c, text, opt.Entities, opt.ParseMode)
	if err != nil {
		return nil, err
	}

	linkPreviewOptions := &LinkPreviewOptions{
		IsDisabled:      opt.DisableWebPagePreview,
		Url:             opt.Url,
		ForceSmallMedia: opt.ForceSmallMedia,
		ForceLargeMedia: opt.ForceLargeMedia,
		ShowAboveText:   opt.ShowAboveText,
	}

	content := &InputMessageText{
		Text:               formattedText,
		LinkPreviewOptions: linkPreviewOptions,
	}

	return c.EditMessageText(chatId, content, messageId, &EditMessageTextOpts{
		ReplyMarkup: opt.ReplyMarkup,
	})
}

// EditCaptionOpts contains optional parameters for EditCaption
type EditCaptionOpts struct {
	ParseMode             string
	Entities              []TextEntity
	ShowCaptionAboveMedia bool
	ReplyMarkup           ReplyMarkup
}

// EditCaption edits the caption of a message
func (c *Client) EditCaption(chatId int64, messageId int64, caption string, opts ...*EditCaptionOpts) (*Message, error) {
	opt := getVariadic(opts, &EditCaptionOpts{})

	if !*c.config.UseMessageDatabase {
		if _, err := c.GetMessage(chatId, messageId); err != nil {
			return nil, err
		}
	}

	formattedText, err := GetFormattedText(c, caption, opt.Entities, opt.ParseMode)
	if err != nil {
		return nil, err
	}

	return c.EditMessageCaption(chatId, messageId, &EditMessageCaptionOpts{
		Caption:               formattedText,
		ReplyMarkup:           opt.ReplyMarkup,
		ShowCaptionAboveMedia: opt.ShowCaptionAboveMedia,
	})
}

// GetSupergroupId returns the supergroup ID from a chat ID
func (c *Client) GetSupergroupId(chatId int64) (int64, error) {
	chat, err := c.GetChat(chatId)
	if err != nil {
		return 0, err
	}

	if chat.Type == nil {
		return 0, nil
	}

	if ct, ok := chat.Type.(*ChatTypeSupergroup); ok {
		return ct.SupergroupId, nil
	}

	return 0, nil
}

// ParseText parses the text using the specified parse mode.
func (c *Client) ParseText(text string, parseMode string) (*FormattedText, error) {
	var mode TextParseMode

	switch strings.ToLower(parseMode) {
	case "markdown":
		mode = &TextParseModeMarkdown{Version: 1}
	case "markdownv2":
		mode = &TextParseModeMarkdown{Version: 2}
	case "html":
		mode = &TextParseModeHTML{}
	default:
		return &FormattedText{Text: text}, nil
	}

	return c.ParseTextEntities(mode, text)
}

// SendChecklistOpts contains optional parameters for SendChecklist
type SendChecklistOpts struct {
	DisableNotification bool
	ProtectContent      bool
	AllowPaidBroadcast  bool
	TopicId             MessageTopic
	Quote               *InputTextQuote
	ReplyTo             InputMessageReplyTo
	ReplyToMessageID    int64
	ReplyMarkup         ReplyMarkup
	EffectId            int64
}

// SendChecklist sends a checklist to chat
func (c *Client) SendChecklist(chatId int64, checklist *InputChecklist, opts ...*SendChecklistOpts) (*Message, error) {
	opt := getVariadic(opts, &SendChecklistOpts{})
	content := &InputMessageChecklist{
		Checklist: checklist,
	}
	return c.sendMessageWithContent(chatId, content, &MessageSendOptions{
		DisableNotification: opt.DisableNotification,
		ProtectContent:      opt.ProtectContent,
		AllowPaidBroadcast:  opt.AllowPaidBroadcast,
		EffectId:            opt.EffectId,
	}, opt.TopicId, opt.Quote, opt.ReplyTo, opt.ReplyToMessageID, opt.ReplyMarkup)
}

// SendContactOpts contains optional parameters for SendContact
type SendContactOpts struct {
	DisableNotification bool
	ProtectContent      bool
	AllowPaidBroadcast  bool
	TopicId             MessageTopic
	Quote               *InputTextQuote
	ReplyTo             InputMessageReplyTo
	ReplyToMessageID    int64
	ReplyMarkup         ReplyMarkup
	EffectId            int64
}

// SendContact sends a contact to chat
func (c *Client) SendContact(chatId int64, contact *Contact, opts ...*SendContactOpts) (*Message, error) {
	opt := getVariadic(opts, &SendContactOpts{})
	content := &InputMessageContact{
		Contact: contact,
	}
	return c.sendMessageWithContent(chatId, content, &MessageSendOptions{
		DisableNotification: opt.DisableNotification,
		ProtectContent:      opt.ProtectContent,
		AllowPaidBroadcast:  opt.AllowPaidBroadcast,
		EffectId:            opt.EffectId,
	}, opt.TopicId, opt.Quote, opt.ReplyTo, opt.ReplyToMessageID, opt.ReplyMarkup)
}

// SendDiceOpts contains optional parameters for SendDice
type SendDiceOpts struct {
	ClearDraft          bool
	DisableNotification bool
	ProtectContent      bool
	AllowPaidBroadcast  bool
	TopicId             MessageTopic
	Quote               *InputTextQuote
	ReplyTo             InputMessageReplyTo
	ReplyToMessageID    int64
	ReplyMarkup         ReplyMarkup
	EffectId            int64
}

// SendDice sends a dice to chat
func (c *Client) SendDice(chatId int64, emoji string, opts ...*SendDiceOpts) (*Message, error) {
	opt := getVariadic(opts, &SendDiceOpts{})
	content := &InputMessageDice{
		Emoji:      emoji,
		ClearDraft: opt.ClearDraft,
	}
	return c.sendMessageWithContent(chatId, content, &MessageSendOptions{
		DisableNotification: opt.DisableNotification,
		ProtectContent:      opt.ProtectContent,
		AllowPaidBroadcast:  opt.AllowPaidBroadcast,
		EffectId:            opt.EffectId,
	}, opt.TopicId, opt.Quote, opt.ReplyTo, opt.ReplyToMessageID, opt.ReplyMarkup)
}

// SendGameOpts contains optional parameters for SendGame
type SendGameOpts struct {
	DisableNotification bool
	ProtectContent      bool
	AllowPaidBroadcast  bool
	TopicId             MessageTopic
	Quote               *InputTextQuote
	ReplyTo             InputMessageReplyTo
	ReplyToMessageID    int64
	ReplyMarkup         ReplyMarkup
	EffectId            int64
}

// SendGame sends a game to chat
func (c *Client) SendGame(chatId int64, botUserId int64, gameShortName string, opts ...*SendGameOpts) (*Message, error) {
	opt := getVariadic(opts, &SendGameOpts{})
	content := &InputMessageGame{
		BotUserId:     botUserId,
		GameShortName: gameShortName,
	}
	return c.sendMessageWithContent(chatId, content, &MessageSendOptions{
		DisableNotification: opt.DisableNotification,
		ProtectContent:      opt.ProtectContent,
		AllowPaidBroadcast:  opt.AllowPaidBroadcast,
		EffectId:            opt.EffectId,
	}, opt.TopicId, opt.Quote, opt.ReplyTo, opt.ReplyToMessageID, opt.ReplyMarkup)
}

// SendInvoiceOpts contains optional parameters for SendInvoice
type SendInvoiceOpts struct {
	PaidMedia           *InputPaidMedia
	PaidMediaCaption    string
	PaidMediaEntities   []TextEntity
	ParseMode           string
	PhotoHeight         int32
	PhotoSize           int32
	PhotoUrl            string
	PhotoWidth          int32
	ProviderData        string
	ProviderToken       string
	StartParameter      string
	DisableNotification bool
	ProtectContent      bool
	AllowPaidBroadcast  bool
	TopicId             MessageTopic
	Quote               *InputTextQuote
	ReplyTo             InputMessageReplyTo
	ReplyToMessageID    int64
	ReplyMarkup         ReplyMarkup
	EffectId            int64
}

// SendInvoice sends an invoice to chat
func (c *Client) SendInvoice(chatId int64, invoice *Invoice, title string, description string, payload []byte, opts ...*SendInvoiceOpts) (*Message, error) {
	opt := getVariadic(opts, &SendInvoiceOpts{})
	paidMediaCaption, err := GetFormattedText(c, opt.PaidMediaCaption, opt.PaidMediaEntities, opt.ParseMode)
	if err != nil {
		return nil, err
	}
	content := &InputMessageInvoice{
		Description:      description,
		Invoice:          invoice,
		PaidMedia:        opt.PaidMedia,
		PaidMediaCaption: paidMediaCaption,
		Payload:          payload,
		PhotoHeight:      opt.PhotoHeight,
		PhotoSize:        opt.PhotoSize,
		PhotoUrl:         opt.PhotoUrl,
		PhotoWidth:       opt.PhotoWidth,
		ProviderData:     opt.ProviderData,
		ProviderToken:    opt.ProviderToken,
		StartParameter:   opt.StartParameter,
		Title:            title,
	}
	return c.sendMessageWithContent(chatId, content, &MessageSendOptions{
		DisableNotification: opt.DisableNotification,
		ProtectContent:      opt.ProtectContent,
		AllowPaidBroadcast:  opt.AllowPaidBroadcast,
		EffectId:            opt.EffectId,
	}, opt.TopicId, opt.Quote, opt.ReplyTo, opt.ReplyToMessageID, opt.ReplyMarkup)
}

// SendLocationOpts contains optional parameters for SendLocation
type SendLocationOpts struct {
	Heading              int32
	LivePeriod           int32
	ProximityAlertRadius int32
	DisableNotification  bool
	ProtectContent       bool
	AllowPaidBroadcast   bool
	TopicId              MessageTopic
	Quote                *InputTextQuote
	ReplyTo              InputMessageReplyTo
	ReplyToMessageID     int64
	ReplyMarkup          ReplyMarkup
	EffectId             int64
}

// SendLocation sends a location to chat
func (c *Client) SendLocation(chatId int64, location *Location, opts ...*SendLocationOpts) (*Message, error) {
	opt := getVariadic(opts, &SendLocationOpts{})
	content := &InputMessageLocation{
		Heading:              opt.Heading,
		LivePeriod:           opt.LivePeriod,
		Location:             location,
		ProximityAlertRadius: opt.ProximityAlertRadius,
	}
	return c.sendMessageWithContent(chatId, content, &MessageSendOptions{
		DisableNotification: opt.DisableNotification,
		ProtectContent:      opt.ProtectContent,
		AllowPaidBroadcast:  opt.AllowPaidBroadcast,
		EffectId:            opt.EffectId,
	}, opt.TopicId, opt.Quote, opt.ReplyTo, opt.ReplyToMessageID, opt.ReplyMarkup)
}

// SendPaidMediaOpts contains optional parameters for SendPaidMedia
type SendPaidMediaOpts struct {
	Caption               string
	CaptionEntities       []TextEntity
	ParseMode             string
	Payload               string
	ShowCaptionAboveMedia bool
	DisableNotification   bool
	ProtectContent        bool
	AllowPaidBroadcast    bool
	TopicId               MessageTopic
	Quote                 *InputTextQuote
	ReplyTo               InputMessageReplyTo
	ReplyToMessageID      int64
	ReplyMarkup           ReplyMarkup
	EffectId              int64
}

// SendPaidMedia sends paid media to chat
func (c *Client) SendPaidMedia(chatId int64, starCount int64, paidMedia []InputPaidMedia, opts ...*SendPaidMediaOpts) (*Message, error) {
	opt := getVariadic(opts, &SendPaidMediaOpts{})
	caption, err := GetFormattedText(c, opt.Caption, opt.CaptionEntities, opt.ParseMode)
	if err != nil {
		return nil, err
	}
	content := &InputMessagePaidMedia{
		Caption:               caption,
		PaidMedia:             paidMedia,
		Payload:               opt.Payload,
		ShowCaptionAboveMedia: opt.ShowCaptionAboveMedia,
		StarCount:             starCount,
	}
	return c.sendMessageWithContent(chatId, content, &MessageSendOptions{
		DisableNotification: opt.DisableNotification,
		ProtectContent:      opt.ProtectContent,
		AllowPaidBroadcast:  opt.AllowPaidBroadcast,
		EffectId:            opt.EffectId,
	}, opt.TopicId, opt.Quote, opt.ReplyTo, opt.ReplyToMessageID, opt.ReplyMarkup)
}

// SendPollOpts contains optional parameters for SendPoll
type SendPollOpts struct {
	AllowsMultipleAnswers  bool
	AllowsRevoting         bool
	CloseDate              int32
	Description            string
	DescriptionEntities    []TextEntity
	ParseMode              string
	HideResultsUntilCloses bool
	IsAnonymous            bool
	IsClosed               bool
	OpenPeriod             int32
	QuestionEntities       []TextEntity
	ShuffleOptions         bool
	Type                   InputPollType
	DisableNotification    bool
	ProtectContent         bool
	AllowPaidBroadcast     bool
	TopicId                MessageTopic
	Quote                  *InputTextQuote
	ReplyTo                InputMessageReplyTo
	ReplyToMessageID       int64
	ReplyMarkup            ReplyMarkup
	EffectId               int64
}

// SendPoll sends a poll to chat
func (c *Client) SendPoll(chatId int64, question string, options []InputPollOption, opts ...*SendPollOpts) (*Message, error) {
	opt := getVariadic(opts, &SendPollOpts{})
	formattedQuestion, err := GetFormattedText(c, question, opt.QuestionEntities, opt.ParseMode)
	if err != nil {
		return nil, err
	}
	description, err := GetFormattedText(c, opt.Description, opt.DescriptionEntities, opt.ParseMode)
	if err != nil {
		return nil, err
	}
	content := &InputMessagePoll{
		AllowsMultipleAnswers:  opt.AllowsMultipleAnswers,
		AllowsRevoting:         opt.AllowsRevoting,
		CloseDate:              opt.CloseDate,
		Description:            description,
		HideResultsUntilCloses: opt.HideResultsUntilCloses,
		IsAnonymous:            opt.IsAnonymous,
		IsClosed:               opt.IsClosed,
		OpenPeriod:             opt.OpenPeriod,
		Options:                options,
		Question:               formattedQuestion,
		ShuffleOptions:         opt.ShuffleOptions,
		Type:                   opt.Type,
	}
	return c.sendMessageWithContent(chatId, content, &MessageSendOptions{
		DisableNotification: opt.DisableNotification,
		ProtectContent:      opt.ProtectContent,
		AllowPaidBroadcast:  opt.AllowPaidBroadcast,
		EffectId:            opt.EffectId,
	}, opt.TopicId, opt.Quote, opt.ReplyTo, opt.ReplyToMessageID, opt.ReplyMarkup)
}

// SendStakeDiceOpts contains optional parameters for SendStakeDice
type SendStakeDiceOpts struct {
	ClearDraft          bool
	DisableNotification bool
	ProtectContent      bool
	AllowPaidBroadcast  bool
	TopicId             MessageTopic
	Quote               *InputTextQuote
	ReplyTo             InputMessageReplyTo
	ReplyToMessageID    int64
	ReplyMarkup         ReplyMarkup
	EffectId            int64
}

// SendStakeDice sends a stake dice to chat
func (c *Client) SendStakeDice(chatId int64, stakeToncoinAmount int64, stateHash string, opts ...*SendStakeDiceOpts) (*Message, error) {
	opt := getVariadic(opts, &SendStakeDiceOpts{})
	content := &InputMessageStakeDice{
		ClearDraft:         opt.ClearDraft,
		StakeToncoinAmount: stakeToncoinAmount,
		StateHash:          stateHash,
	}
	return c.sendMessageWithContent(chatId, content, &MessageSendOptions{
		DisableNotification: opt.DisableNotification,
		ProtectContent:      opt.ProtectContent,
		AllowPaidBroadcast:  opt.AllowPaidBroadcast,
		EffectId:            opt.EffectId,
	}, opt.TopicId, opt.Quote, opt.ReplyTo, opt.ReplyToMessageID, opt.ReplyMarkup)
}

// SendStoryOpts contains optional parameters for SendStory
type SendStoryOpts struct {
	DisableNotification bool
	ProtectContent      bool
	AllowPaidBroadcast  bool
	TopicId             MessageTopic
	Quote               *InputTextQuote
	ReplyTo             InputMessageReplyTo
	ReplyToMessageID    int64
	ReplyMarkup         ReplyMarkup
	EffectId            int64
}

// SendStory sends a story to chat
func (c *Client) SendStory(chatId int64, storyPosterChatId int64, storyId int32, opts ...*SendStoryOpts) (*Message, error) {
	opt := getVariadic(opts, &SendStoryOpts{})
	content := &InputMessageStory{
		StoryId:           storyId,
		StoryPosterChatId: storyPosterChatId,
	}
	return c.sendMessageWithContent(chatId, content, &MessageSendOptions{
		DisableNotification: opt.DisableNotification,
		ProtectContent:      opt.ProtectContent,
		AllowPaidBroadcast:  opt.AllowPaidBroadcast,
		EffectId:            opt.EffectId,
	}, opt.TopicId, opt.Quote, opt.ReplyTo, opt.ReplyToMessageID, opt.ReplyMarkup)
}

// SendVenueOpts contains optional parameters for SendVenue
type SendVenueOpts struct {
	DisableNotification bool
	ProtectContent      bool
	AllowPaidBroadcast  bool
	TopicId             MessageTopic
	Quote               *InputTextQuote
	ReplyTo             InputMessageReplyTo
	ReplyToMessageID    int64
	ReplyMarkup         ReplyMarkup
	EffectId            int64
}

// SendVenue sends a venue to chat
func (c *Client) SendVenue(chatId int64, venue *Venue, opts ...*SendVenueOpts) (*Message, error) {
	opt := getVariadic(opts, &SendVenueOpts{})
	content := &InputMessageVenue{
		Venue: venue,
	}
	return c.sendMessageWithContent(chatId, content, &MessageSendOptions{
		DisableNotification: opt.DisableNotification,
		ProtectContent:      opt.ProtectContent,
		AllowPaidBroadcast:  opt.AllowPaidBroadcast,
		EffectId:            opt.EffectId,
	}, opt.TopicId, opt.Quote, opt.ReplyTo, opt.ReplyToMessageID, opt.ReplyMarkup)
}
