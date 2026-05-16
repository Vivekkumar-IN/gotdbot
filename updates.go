package gotdbot

import (
	"regexp"
	"strings"
	"sync/atomic"
)

// --- Update Handling Types ---

type (
	Handle interface {
		SetGroup(group int) Handle
		GetGroup() int
		SetPriority(priority int) Handle
		GetPriority() int
		Check(*Client, TlObject) bool
		Execute(*Client, TlObject) error
		GetType() UpdateType
	}

	HandlerFunc[T any] func(*Client, *T) error

	RawHandlerFunc func(*Client, TlObject) error

	handle[T any] struct {
		client   *Client
		handler  HandlerFunc[T]
		filters  []Filter
		tp       UpdateType
		group    atomic.Int32
		priority atomic.Int32
	}
)

const UpdateTypeRaw UpdateType = "__raw__"

func (h *handle[T]) SetGroup(group int) Handle {
	h.group.Store(int32(group))
	if h.client != nil {
		h.client.sortHandlers(h.tp)
	}
	return h
}

func (h *handle[T]) GetGroup() int {
	return int(h.group.Load())
}

func (h *handle[T]) SetPriority(priority int) Handle {
	h.priority.Store(int32(priority))
	if h.client != nil {
		h.client.sortHandlers(h.tp)
	}
	return h
}

func (h *handle[T]) GetPriority() int {
	return int(h.priority.Load())
}

func (h *handle[T]) GetType() UpdateType {
	return h.tp
}

func (h *handle[T]) applyDefaultFilters() {
	_, ok := any(new(T)).(*UpdateNewMessage)
	if !ok {
		return
	}

	if h.hasOutgoingFilter() {
		return
	}

	h.filters = append(h.filters, FilterIncoming)
}

func (h *handle[T]) hasOutgoingFilter() bool {
	for _, f := range h.filters {
		if h.isOutgoingFilter(f) {
			return true
		}
	}
	return false
}

func (h *handle[T]) isOutgoingFilter(f Filter) bool {
	if of, ok := f.(interface{ IsOutgoing() bool }); ok {
		return of.IsOutgoing()
	}
	return false
}

func (h *handle[T]) Check(c *Client, update TlObject) bool {
	u, ok := any(update).(*T)
	if !ok {
		return false
	}

	for _, f := range h.filters {
		var match bool
		switch upd := any(u).(type) {
		case *UpdateNewMessage:
			match = f.Check(c, upd)
		case *UpdateNewCallbackQuery:
			match = f.CheckCB(c, upd)
		default:
			match = f.CheckAny(c, any(upd).(TlObject))
		}
		if !match {
			return false
		}
	}

	return true
}

func (h *handle[T]) Execute(c *Client, update TlObject) error {
	return h.handler(c, any(update).(*T))
}

type rawHandle struct {
	client   *Client
	handler  RawHandlerFunc
	filters  []Filter
	group    atomic.Int32
	priority atomic.Int32
}

func (h *rawHandle) SetGroup(group int) Handle {
	h.group.Store(int32(group))
	if h.client != nil {
		h.client.sortHandlers(UpdateTypeRaw)
	}
	return h
}

func (h *rawHandle) GetGroup() int {
	return int(h.group.Load())
}

func (h *rawHandle) SetPriority(priority int) Handle {
	h.priority.Store(int32(priority))
	if h.client != nil {
		h.client.sortHandlers(UpdateTypeRaw)
	}
	return h
}

func (h *rawHandle) GetPriority() int {
	return int(h.priority.Load())
}

func (h *rawHandle) GetType() UpdateType {
	return UpdateTypeRaw
}

func (h *rawHandle) Check(c *Client, update TlObject) bool {
	for _, f := range h.filters {
		if !f.CheckAny(c, update) {
			return false
		}
	}
	return true
}

func (h *rawHandle) Execute(c *Client, update TlObject) error {
	return h.handler(c, update)
}

// --- Filter System ---

type Filter interface {
	Check(c *Client, u *UpdateNewMessage) bool
	CheckCB(c *Client, u *UpdateNewCallbackQuery) bool
	CheckAny(c *Client, u TlObject) bool
	And(other Filter) Filter
	Or(other Filter) Filter
	Not() Filter
}

type filterFunc struct {
	check    func(*Client, *UpdateNewMessage) bool
	checkCB  func(*Client, *UpdateNewCallbackQuery) bool
	checkAny func(*Client, TlObject) bool
}

func (f *filterFunc) Check(c *Client, u *UpdateNewMessage) bool {
	if f.check != nil {
		return f.check(c, u)
	}
	return true
}

func (f *filterFunc) CheckCB(c *Client, u *UpdateNewCallbackQuery) bool {
	if f.checkCB != nil {
		return f.checkCB(c, u)
	}
	return true
}

func (f *filterFunc) CheckAny(c *Client, u TlObject) bool {
	if f.checkAny != nil {
		return f.checkAny(c, u)
	}
	switch upd := u.(type) {
	case *UpdateNewMessage:
		return f.Check(c, upd)
	case *UpdateNewCallbackQuery:
		return f.CheckCB(c, upd)
	}
	return f.check == nil && f.checkCB == nil
}

func (f *filterFunc) And(other Filter) Filter {
	return &andFilter{filters: []Filter{f, other}}
}

func (f *filterFunc) Or(other Filter) Filter {
	return &orFilter{filters: []Filter{f, other}}
}

func (f *filterFunc) Not() Filter {
	return &notFilter{filter: f}
}

type andFilter struct {
	filters []Filter
}

func (f *andFilter) Check(c *Client, u *UpdateNewMessage) bool {
	for _, filter := range f.filters {
		if !filter.Check(c, u) {
			return false
		}
	}
	return true
}

func (f *andFilter) CheckCB(c *Client, u *UpdateNewCallbackQuery) bool {
	for _, filter := range f.filters {
		if !filter.CheckCB(c, u) {
			return false
		}
	}
	return true
}

func (f *andFilter) CheckAny(c *Client, u TlObject) bool {
	for _, filter := range f.filters {
		if !filter.CheckAny(c, u) {
			return false
		}
	}
	return true
}

func (f *andFilter) And(other Filter) Filter {
	f.filters = append(f.filters, other)
	return f
}

func (f *andFilter) Or(other Filter) Filter {
	return &orFilter{filters: []Filter{f, other}}
}

func (f *andFilter) Not() Filter {
	return &notFilter{filter: f}
}

func (f *andFilter) IsOutgoing() bool {
	for _, filter := range f.filters {
		if of, ok := filter.(interface{ IsOutgoing() bool }); ok && of.IsOutgoing() {
			return true
		}
	}
	return false
}

type orFilter struct {
	filters []Filter
}

func (f *orFilter) Check(c *Client, u *UpdateNewMessage) bool {
	for _, filter := range f.filters {
		if filter.Check(c, u) {
			return true
		}
	}
	return false
}

func (f *orFilter) CheckCB(c *Client, u *UpdateNewCallbackQuery) bool {
	for _, filter := range f.filters {
		if filter.CheckCB(c, u) {
			return true
		}
	}
	return false
}

func (f *orFilter) CheckAny(c *Client, u TlObject) bool {
	for _, filter := range f.filters {
		if filter.CheckAny(c, u) {
			return true
		}
	}
	return false
}

func (f *orFilter) And(other Filter) Filter {
	return &andFilter{filters: []Filter{f, other}}
}

func (f *orFilter) Or(other Filter) Filter {
	f.filters = append(f.filters, other)
	return f
}

func (f *orFilter) Not() Filter {
	return &notFilter{filter: f}
}

func (f *orFilter) IsOutgoing() bool {
	for _, filter := range f.filters {
		if of, ok := filter.(interface{ IsOutgoing() bool }); ok && of.IsOutgoing() {
			return true
		}
	}
	return false
}

type notFilter struct {
	filter Filter
}

func (f *notFilter) Check(c *Client, u *UpdateNewMessage) bool {
	return !f.filter.Check(c, u)
}

func (f *notFilter) CheckCB(c *Client, u *UpdateNewCallbackQuery) bool {
	return !f.filter.CheckCB(c, u)
}

func (f *notFilter) CheckAny(c *Client, u TlObject) bool {
	return !f.filter.CheckAny(c, u)
}

func (f *notFilter) And(other Filter) Filter {
	return &andFilter{filters: []Filter{f, other}}
}

func (f *notFilter) Or(other Filter) Filter {
	return &orFilter{filters: []Filter{f, other}}
}

func (f *notFilter) Not() Filter {
	return f.filter
}

// --- Default Filters ---

type outgoingFilter struct {
	Filter
}

func (f *outgoingFilter) IsOutgoing() bool {
	return true
}

var (
	FilterAll Filter = &filterFunc{}

	FilterIncoming Filter = &filterFunc{
		check: func(_ *Client, u *UpdateNewMessage) bool {
			return u.Message != nil && !u.Message.IsOutgoing
		},
	}

	FilterOutgoing Filter = &outgoingFilter{
		Filter: &filterFunc{
			check: func(_ *Client, u *UpdateNewMessage) bool {
				return u.Message != nil && u.Message.IsOutgoing
			},
		},
	}

	FilterPrivate Filter = &filterFunc{
		check: func(_ *Client, u *UpdateNewMessage) bool {
			return u.Message != nil && u.Message.ChatId > 0
		},
		checkCB: func(_ *Client, u *UpdateNewCallbackQuery) bool {
			return u.ChatId > 0
		},
		checkAny: func(_ *Client, u TlObject) bool {
			id := ExtractChatID(u)
			return id > 0
		},
	}

	FilterGroup Filter = &filterFunc{
		check: func(_ *Client, u *UpdateNewMessage) bool {
			return u.Message != nil && u.Message.IsGroup()
		},
		checkCB: func(_ *Client, u *UpdateNewCallbackQuery) bool {
			return u.ChatId < 0 && !isSupergroupOrChannelID(u.ChatId)
		},
		checkAny: func(_ *Client, u TlObject) bool {
			id := ExtractChatID(u)
			return id < 0 && !isSupergroupOrChannelID(id)
		},
	}

	FilterChannel Filter = &filterFunc{
		check: func(_ *Client, u *UpdateNewMessage) bool {
			return u.Message != nil && u.Message.IsSupergroupOrChannel()
		},
		checkCB: func(_ *Client, u *UpdateNewCallbackQuery) bool {
			return isSupergroupOrChannelID(u.ChatId)
		},
		checkAny: func(_ *Client, u TlObject) bool {
			id := ExtractChatID(u)
			return isSupergroupOrChannelID(id)
		},
	}

	FilterReply Filter = &filterFunc{
		check: func(_ *Client, u *UpdateNewMessage) bool {
			return u.Message != nil && u.Message.ReplyTo != nil
		},
		checkAny: func(_ *Client, u TlObject) bool {
			if upd, ok := u.(*UpdateNewMessage); ok {
				return upd.Message != nil && upd.Message.ReplyTo != nil
			}
			return false
		},
	}

	FilterForward Filter = &filterFunc{
		check: func(_ *Client, u *UpdateNewMessage) bool {
			return u.Message != nil && u.Message.ForwardInfo != nil
		},
		checkAny: func(_ *Client, u TlObject) bool {
			if upd, ok := u.(*UpdateNewMessage); ok {
				return upd.Message != nil && upd.Message.ForwardInfo != nil
			}
			return false
		},
	}

	FilterEdited Filter = &filterFunc{
		checkAny: func(_ *Client, u TlObject) bool {
			_, ok := u.(*UpdateMessageEdited)
			return ok
		},
	}

	FilterText Filter = &filterFunc{
		check: func(_ *Client, u *UpdateNewMessage) bool {
			if u.Message == nil {
				return false
			}
			return u.Message.Text() != ""
		},
	}

	FilterPhoto Filter = &filterFunc{
		check: func(_ *Client, u *UpdateNewMessage) bool {
			if u.Message == nil || u.Message.Content == nil {
				return false
			}
			_, ok := u.Message.Content.(*MessagePhoto)
			return ok
		},
	}

	FilterVideo Filter = &filterFunc{
		check: func(_ *Client, u *UpdateNewMessage) bool {
			if u.Message == nil || u.Message.Content == nil {
				return false
			}
			_, ok := u.Message.Content.(*MessageVideo)
			return ok
		},
	}

	FilterAnimation Filter = &filterFunc{
		check: func(_ *Client, u *UpdateNewMessage) bool {
			if u.Message == nil || u.Message.Content == nil {
				return false
			}
			_, ok := u.Message.Content.(*MessageAnimation)
			return ok
		},
	}

	FilterAudio Filter = &filterFunc{
		check: func(_ *Client, u *UpdateNewMessage) bool {
			if u.Message == nil || u.Message.Content == nil {
				return false
			}
			_, ok := u.Message.Content.(*MessageAudio)
			return ok
		},
	}

	FilterDocument Filter = &filterFunc{
		check: func(_ *Client, u *UpdateNewMessage) bool {
			if u.Message == nil || u.Message.Content == nil {
				return false
			}
			_, ok := u.Message.Content.(*MessageDocument)
			return ok
		},
	}

	FilterSticker Filter = &filterFunc{
		check: func(_ *Client, u *UpdateNewMessage) bool {
			if u.Message == nil || u.Message.Content == nil {
				return false
			}
			_, ok := u.Message.Content.(*MessageSticker)
			return ok
		},
	}

	FilterVideoNote Filter = &filterFunc{
		check: func(_ *Client, u *UpdateNewMessage) bool {
			if u.Message == nil || u.Message.Content == nil {
				return false
			}
			_, ok := u.Message.Content.(*MessageVideoNote)
			return ok
		},
	}

	FilterVoiceNote Filter = &filterFunc{
		check: func(_ *Client, u *UpdateNewMessage) bool {
			if u.Message == nil || u.Message.Content == nil {
				return false
			}
			_, ok := u.Message.Content.(*MessageVoiceNote)
			return ok
		},
	}
)

func FilterChatID(id int64) Filter {
	return &filterFunc{
		check: func(_ *Client, u *UpdateNewMessage) bool {
			return u.Message != nil && u.Message.ChatId == id
		},
		checkCB: func(_ *Client, u *UpdateNewCallbackQuery) bool {
			return u.ChatId == id
		},
		checkAny: func(_ *Client, u TlObject) bool {
			return ExtractChatID(u) == id
		},
	}
}

func FilterSenderID(id int64) Filter {
	return &filterFunc{
		check: func(_ *Client, u *UpdateNewMessage) bool {
			return u.Message != nil && u.Message.SenderID() == id
		},
		checkCB: func(_ *Client, u *UpdateNewCallbackQuery) bool {
			return u.SenderUserId == id
		},
		checkAny: func(_ *Client, u TlObject) bool {
			return ExtractSenderID(u) == id
		},
	}
}

func FilterCommand(command string) Filter {
	return &filterFunc{
		check: func(c *Client, u *UpdateNewMessage) bool {
			if u.Message == nil {
				return false
			}
			text := u.Message.Text()
			if text == "" {
				return false
			}

			prefixes := c.config.CommandPrefixes
			if prefixes == "" {
				prefixes = "/"
			}

			for _, prefix := range prefixes {
				cmd := string(prefix) + command
				if text == cmd || strings.HasPrefix(text, cmd+" ") {
					return true
				}

				if c.Me != nil && c.Me.Usernames != nil {
					for _, username := range c.Me.Usernames.ActiveUsernames {
						atCmd := cmd + "@" + username
						if text == atCmd || strings.HasPrefix(text, atCmd+" ") {
							return true
						}
					}
				}
			}
			return false
		},
	}
}

func FilterContains(match string) Filter {
	return &filterFunc{
		check: func(_ *Client, u *UpdateNewMessage) bool {
			if u.Message == nil {
				return false
			}
			return strings.Contains(u.Message.GetText(), match)
		},
		checkCB: func(_ *Client, u *UpdateNewCallbackQuery) bool {
			return strings.Contains(u.DataString(), match)
		},
	}
}

func FilterPrefix(prefix string) Filter {
	return &filterFunc{
		check: func(_ *Client, u *UpdateNewMessage) bool {
			if u.Message == nil {
				return false
			}
			return strings.HasPrefix(u.Message.GetText(), prefix)
		},
		checkCB: func(_ *Client, u *UpdateNewCallbackQuery) bool {
			return strings.HasPrefix(u.DataString(), prefix)
		},
	}
}

func FilterSuffix(suffix string) Filter {
	return &filterFunc{
		checkCB: func(_ *Client, u *UpdateNewCallbackQuery) bool {
			return strings.HasSuffix(u.DataString(), suffix)
		},
	}
}

func FilterEqual(match string) Filter {
	return &filterFunc{
		check: func(_ *Client, u *UpdateNewMessage) bool {
			if u.Message == nil {
				return false
			}
			return u.Message.GetText() == match
		},
		checkCB: func(_ *Client, u *UpdateNewCallbackQuery) bool {
			return u.DataString() == match
		},
	}
}

func FilterRegex(pattern string) Filter {
	reg, err := regexp.Compile(pattern)
	if err != nil {
		return &filterFunc{
			check:   func(_ *Client, _ *UpdateNewMessage) bool { return false },
			checkCB: func(_ *Client, _ *UpdateNewCallbackQuery) bool { return false },
		}
	}
	return &filterFunc{
		check: func(_ *Client, u *UpdateNewMessage) bool {
			if u.Message == nil {
				return false
			}
			return reg.MatchString(u.Message.GetText())
		},
		checkCB: func(_ *Client, u *UpdateNewCallbackQuery) bool {
			data := u.CallbackData()
			if data == nil {
				return false
			}
			return reg.Match(data)
		},
	}
}
