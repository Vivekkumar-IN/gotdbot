package gotdbot

import (
	"time"
)

// WaitMessageOpts holds optional parameters for Ask.
type WaitMessageOpts struct {
	Filter             Filter
	CancellationFilter Filter
	Timeout            time.Duration
}

// Ask waits for a new message in the specified chat.
func (c *Client) Ask(chatId int64, opts *WaitMessageOpts) (*Message, error) {
	if opts == nil {
		opts = &WaitMessageOpts{Timeout: 1 * time.Minute}
	}

	filter := func(client *Client, update TlObject) bool {
		u, ok := update.(*UpdateNewMessage)
		if !ok || u == nil {
			return false
		}

		msg := u.Message
		if msg.ChatId != chatId {
			return false
		}

		if opts.CancellationFilter != nil && opts.CancellationFilter.Check(client, u) {
			return true
		}

		if opts.Filter != nil && !opts.Filter.Check(client, u) {
			return false
		}

		return true
	}

	raw, err := c.WaitFor(filter, opts.Timeout)
	if err != nil {
		return nil, err
	}

	u := raw.(*UpdateNewMessage)
	msg := u.Message

	if opts.CancellationFilter != nil && opts.CancellationFilter.Check(c, u) {
		return nil, ConversationCancelled
	}

	return msg, nil
}
