package gotdbot

//go:generate go run ./internal/gen

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"runtime"

	"os/signal"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Vivekkumar-IN/gotdbot/internal/qrcode"
	"github.com/Vivekkumar-IN/gotdbot/internal/tdjson"
)

type Client struct {
	manager  *ClientManager
	clientID int

	apiID    int32
	apiHash  string
	botToken string
	// phoneNumber is set if the user is logging in with a phone number.
	phoneNumber string
	config      *ClientOpts
	Logger      *slog.Logger

	updates chan TlObject
	stop    chan struct{}
	closed  chan struct{}
	wg      sync.WaitGroup

	pendingRequests sync.Map // map[string]chan TlObject
	pendingMessages sync.Map // map[string]chan TlObject

	// Auth state management
	authErrorChan chan error
	isAuthorized  bool
	startOnce     sync.Once
	closeOnce     sync.Once

	handleMu sync.RWMutex
	handlers map[UpdateType][]Handle

	waiters     map[string]*Waiter
	waiterCount int64
	wMu         sync.RWMutex

	Options   map[string]interface{}
	optionsMu sync.RWMutex

	// PanicHandler handles panics during update processing.
	PanicHandler func(client *Client, update TlObject, r any)

	// ErrorHandler handles errors returned by handlers.
	ErrorHandler func(client *Client, update TlObject, err error) error

	// Me is the bot's User info, as returned by client.GetMe.
	// Populated when authorization is ready.
	Me *User
}

func NewClient(apiID int32, apiHash, tokenOrPhone string, opts ...*ClientOpts) (*Client, error) {
	tokenOrPhone = strings.TrimSpace(tokenOrPhone)
	config := getVariadic(opts, &ClientOpts{})

	if config.UseFileDatabase == nil {
		config.UseFileDatabase = Bool(true)
	}
	if config.UseChatInfoDatabase == nil {
		config.UseChatInfoDatabase = Bool(true)
	}
	if config.UseMessageDatabase == nil {
		config.UseMessageDatabase = Bool(true)
	}
	if config.SystemLanguageCode == "" {
		config.SystemLanguageCode = "en"
	}
	if config.DeviceModel == "" {
		config.DeviceModel = "Gotdbot"
	}
	if config.SystemVersion == "" {
		config.SystemVersion = runtime.GOOS
	}
	if config.ApplicationVersion == "" {
		config.ApplicationVersion = "Gotdbot " + Version
	}
	if config.DatabaseDirectory == "" {
		config.DatabaseDirectory = "database"
	}
	if config.Logger == nil {
		config.Logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
	if config.AuthorizationTimeout == 0 {
		config.AuthorizationTimeout = 60 * time.Second
	}

	if config.LogVerbosityLevel == 0 {
		config.LogVerbosityLevel = 2
	}

	if config.AutoRetry == nil {
		config.AutoRetry = &AutoRetry{}
	}

	if config.CommandPrefixes == "" {
		config.CommandPrefixes = "/"
	}

	if err := tdjson.Init(config.LibraryPath, TDLibVersion); err != nil {
		return nil, err
	}

	botTokenRegex := regexp.MustCompile(`^\d+:[a-zA-Z0-9_-]+$`)
	var botToken, phoneNumber string
	if botTokenRegex.MatchString(tokenOrPhone) {
		botToken = tokenOrPhone
	} else {
		phoneNumber = tokenOrPhone
	}

	c := &Client{
		clientID:      tdjson.CreateClientID(),
		apiID:         apiID,
		apiHash:       apiHash,
		botToken:      botToken,
		phoneNumber:   phoneNumber,
		config:        config,
		Logger:        config.Logger,
		updates:       make(chan TlObject, 1000),
		stop:          make(chan struct{}),
		closed:        make(chan struct{}),
		authErrorChan: make(chan error, 1),
		handlers:      make(map[UpdateType][]Handle),
		waiters:       make(map[string]*Waiter),
		Options:       make(map[string]interface{}),
	}

	if config.PanicHandler != nil {
		c.PanicHandler = config.PanicHandler
	} else {
		c.PanicHandler = func(client *Client, update TlObject, r any) {
			c.Logger.Error("Panic in update handler", "panic", r, "stack", string(debug.Stack()))
		}
	}

	if config.ErrorHandler != nil {
		c.ErrorHandler = config.ErrorHandler
	} else {
		c.ErrorHandler = func(client *Client, update TlObject, err error) error {
			c.Logger.Error("Handler error", "error", err)
			return nil
		}
	}

	c.AddAuthorizationStateHandler(c.authHandler).SetGroup(-999)
	c.AddMessageSendSucceededHandler(c.messageSendSucceededHandler).SetGroup(-998)
	c.AddMessageSendFailedHandler(c.messageSendFailedHandler).SetGroup(-997)
	c.AddOptionHandler(c.optionHandler).SetGroup(-996)
	c.AddUserHandler(c.updateUserHandler).SetGroup(-995)
	c.AddConnectionStateHandler(c.connectionStateHandler).SetGroup(-2)
	return c, nil
}

// initClient initializes the client's internal state and processor loop exactly once.
func (c *Client) initClient() {
	tdjson.Send(c.clientID, `{"@type": "getOption", "name": "version"}`)

	c.startOnce.Do(func() {
		if c.manager == nil {
			m := GetDefaultManager(c.config.LibraryPath)
			m.AddClient(c)
		}

		c.wg.Add(1)
		go c.processor()
	})
}

// Start initializes the client and blocks until authorization is successful or fails.
func (c *Client) Start() error {
	c.initClient()

	select {
	case err := <-c.authErrorChan:
		return err
	case <-time.After(c.config.AuthorizationTimeout):
		return fmt.Errorf("authorization timeout")
	}
}

// SendDummyRequest sends a dummy request to TDLib to initialize its state.
// This is useful if you need to call unauthenticated TDLib methods before calling Start().
func (c *Client) SendDummyRequest() {
	c.initClient()
}

// Idle blocks the current goroutine until a SIGINT or SIGTERM signal is received.
func (c *Client) Idle() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	c.Close()
}

// Close Closes the TDLib instance. All databases will be flushed to disk and properly closed. After the close completes, updateAuthorizationState with authorizationStateClosed will be sent. Can be called before initialization
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		_, _ = c.Send(&Close{})

		select {
		case <-c.closed:
		case <-time.After(5 * time.Second):
			c.Logger.Warn("Timeout waiting for TDLib to close")
		}

		close(c.stop)
		c.wg.Wait()
	})
}

func (c *Client) processor() {
	defer c.wg.Done()
	for {
		select {
		case <-c.stop:
			return
		case update := <-c.updates:
			c.processUpdate(update)
		}
	}
}

func (c *Client) authHandler(client *Client, authState *UpdateAuthorizationState) error {
	c.Logger.Debug("Authorization state update", "state", authState.AuthorizationState.GetType())

	switch authState.AuthorizationState.GetType() {
	case "authorizationStateWaitTdlibParameters":
		if c.config.TDLibOptions != nil {
			c.config.TDLibOptions.forEachSet(func(k string, v interface{}) {
				if opt := toOptionValue(v); opt != nil {
					err := c.SetOption(k, &SetOptionOpts{Value: opt})
					if err != nil {
						c.Logger.Error("Error setting option", "option", k, "error", err)
					}
				}
			})
		}

		err := c.SetTdlibParameters(
			c.apiHash,
			c.apiID,
			c.config.ApplicationVersion,
			c.config.DatabaseDirectory,
			[]byte(c.config.DatabaseEncryptionKey),
			c.config.DeviceModel,
			c.config.FilesDirectory,
			c.config.SystemLanguageCode,
			c.config.SystemVersion,
			&SetTdlibParametersOpts{
				UseChatInfoDatabase: *c.config.UseChatInfoDatabase,
				UseFileDatabase:     *c.config.UseFileDatabase,
				UseMessageDatabase:  *c.config.UseMessageDatabase,
				UseSecretChats:      c.config.UseSecretChats,
				UseTestDc:           c.config.UseTestDC,
			},
		)
		if err != nil {
			c.Logger.Error("Error setting tdlib parameters", "error", err)
			c.authErrorChan <- err
		}

	case "authorizationStateWaitPhoneNumber":
		if c.botToken != "" {
			err := c.CheckAuthenticationBotToken(c.botToken)
			if err != nil {
				c.Logger.Error("Error checking bot token", "error", err)
				c.authErrorChan <- err
			}
		} else {
			// User login
			if c.config.QrMode {
				err := c.RequestQrCodeAuthentication(nil)
				if err != nil {
					c.Logger.Error("Error requesting QR code", "error", err)
					c.authErrorChan <- err
				}
			} else if c.phoneNumber != "" {
				err := c.SetAuthenticationPhoneNumber(c.phoneNumber, nil)
				if err != nil {
					c.Logger.Error("Error setting phone number", "error", err)
					c.authErrorChan <- err
				}
			} else {
				fmt.Print("Enter phone number: ")
				reader := bufio.NewReader(os.Stdin)
				phone, _ := reader.ReadString('\n')
				phone = strings.TrimSpace(phone)
				err := c.SetAuthenticationPhoneNumber(phone, nil)
				if err != nil {
					c.Logger.Error("Error setting phone number", "error", err)
					c.authErrorChan <- err
				}
			}
		}

	case "authorizationStateWaitOtherDeviceConfirmation":
		link := authState.AuthorizationState.(*AuthorizationStateWaitOtherDeviceConfirmation).Link
		fmt.Printf("Scan the QR code below: or open the link in Telegram: %s\n", link)
		q, err := qrcode.NewQRCode(link)
		if err != nil {
			fmt.Printf("Failed to generate QR: %v\nLink: %s\n", err, link)
		} else {
			fmt.Println(q.ToSmallString(false))
		}

	case "authorizationStateWaitCode":
		codeInfo := authState.AuthorizationState.(*AuthorizationStateWaitCode).CodeInfo
		codeType := codeInfo.Type.GetType()
		codeType = strings.TrimPrefix(codeType, "authenticationCodeType")
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Printf("Enter the code received via %s: ", codeType)
			code, _ := reader.ReadString('\n')
			code = strings.TrimSpace(code)
			if code == "" {
				continue
			}
			err := c.CheckAuthenticationCode(code)
			if err != nil {
				fmt.Printf("Error checking code: %v\n", err)
				continue
			}
			break
		}

	case "authorizationStateWaitPassword":
		hint := authState.AuthorizationState.(*AuthorizationStateWaitPassword).PasswordHint
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Printf("Enter your 2FA password (hint: %s): ", hint)
			password, _ := reader.ReadString('\n')
			password = strings.TrimSpace(password)
			if password == "" {
				continue
			}
			err := c.CheckAuthenticationPassword(password)
			if err != nil {
				fmt.Printf("Error checking password: %v\n", err)
				continue
			}
			break
		}

	case "authorizationStateWaitRegistration":
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter first name: ")
		firstName, _ := reader.ReadString('\n')
		firstName = strings.TrimSpace(firstName)
		fmt.Print("Enter last name: ")
		lastName, _ := reader.ReadString('\n')
		lastName = strings.TrimSpace(lastName)

		err := c.RegisterUser(firstName, lastName, nil)
		if err != nil {
			c.Logger.Error("Error registering user", "error", err)
			c.authErrorChan <- err
		}
	case "authorizationStateWaitPremiumPurchase":
		c.Logger.Info("Account is limited and requires Telegram Premium. Please purchase Telegram Premium to continue.")
		c.authErrorChan <- WaitPremiumPurchase
	case "authorizationStateReady":
		c.isAuthorized = true
		me, err := c.GetMe()
		if err != nil {
			c.Logger.Error("Failed to get me", "error", err)
			c.authErrorChan <- err
			return nil
		}

		c.Me = me

		username := ""
		if me.Usernames != nil && len(me.Usernames.ActiveUsernames) > 0 {
			username = me.Usernames.ActiveUsernames[0]
		}
		c.Logger.Info("Logged in", "user_id", me.Id, "username", username)

		select {
		case c.authErrorChan <- nil:
		default:
		}

	case "authorizationStateClosed":
		if !c.isAuthorized {
			c.authErrorChan <- fmt.Errorf("authorization closed unexpectedly")
		}
		select {
		case <-c.closed:
		default:
			close(c.closed)
		}
	}
	return nil
}

func (c *Client) connectionStateHandler(client *Client, u *UpdateConnectionState) error {
	state := u.State.GetType()
	state = strings.TrimPrefix(state, "connectionState")
	c.Logger.Info("Connection state changed", "state", state)
	return nil
}

func (c *Client) messageSendSucceededHandler(client *Client, u *UpdateMessageSendSucceeded) error {
	key := fmt.Sprintf("%d:%d", u.Message.ChatId, u.OldMessageId)
	if ch, ok := c.pendingMessages.Load(key); ok {
		ch.(chan TlObject) <- u
		c.pendingMessages.Delete(key)
	}

	return nil
}

func (c *Client) messageSendFailedHandler(client *Client, u *UpdateMessageSendFailed) error {
	key := fmt.Sprintf("%d:%d", u.Message.ChatId, u.OldMessageId)
	if ch, ok := c.pendingMessages.Load(key); ok {
		ch.(chan TlObject) <- u
		c.pendingMessages.Delete(key)
	}
	return nil
}

func (c *Client) optionHandler(_ *Client, u *UpdateOption) error {
	c.optionsMu.Lock()
	defer c.optionsMu.Unlock()

	switch v := u.Value.(type) {
	case *OptionValueBoolean:
		c.Options[u.Name] = v.Value
	case *OptionValueInteger:
		c.Options[u.Name] = v.Value
	case *OptionValueString:
		c.Options[u.Name] = v.Value
	case *OptionValueEmpty:
		c.Options[u.Name] = nil
	}

	return nil
}

func (c *Client) updateUserHandler(_ *Client, u *UpdateUser) error {
	if !c.isAuthorized || c.Me == nil {
		return nil
	}

	if u.User == nil || u.User.Id != c.Me.Id {
		return nil
	}

	c.Me = u.User

	return nil
}

func (c *Client) Send(req TlObject) (TlObject, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	var tmp map[string]interface{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return nil, err
	}

	reqType, _ := tmp["@type"].(string)
	reqType = strings.ToLower(reqType)

	isChatAttemptedLoad := reqType == "getchat"
	isMessageAttemptedLoad := reqType == "getmessage" || reqType == "getmessagelocally" || reqType == "getrepliedmessage" || reqType == "getcallbackquerymessage"

	for {
		extra := fmt.Sprintf("%d", time.Now().UnixNano())
		tmp["@extra"] = extra

		sendData, err := json.Marshal(tmp)
		if err != nil {
			return nil, err
		}

		ch := make(chan TlObject, 1)
		c.pendingRequests.Store(extra, ch)

		tdjson.Send(c.clientID, string(sendData))

		var result TlObject
		var resultErr error

		select {
		case res := <-ch:
			if res.GetType() == "error" {
				if errObj, ok := res.(*Error); ok {
					resultErr = errObj
					result = res
				}
			} else {
				result = res
			}
		case <-time.After(30 * time.Second):
			c.pendingRequests.Delete(extra)
			return nil, SendTimeout
		}

		if resultErr == nil {
			return result, nil
		}

		errObj := resultErr.(*Error)

		if c.handleAutoRetry(tmp, errObj, &isChatAttemptedLoad, &isMessageAttemptedLoad) {
			continue
		}

		return nil, resultErr
	}
}

func (c *Client) handleAutoRetry(req map[string]interface{}, errObj *Error, isChatAttemptedLoad, isMessageAttemptedLoad *bool) bool {
	if c.config.AutoRetry == nil {
		return false
	}

	if errObj.Code == 429 && c.config.AutoRetry.MaxFloodWait > 0 {
		prefix := "Too Many Requests: retry after "
		if strings.HasPrefix(errObj.Message, prefix) {
			seconds, err := strconv.Atoi(strings.TrimPrefix(errObj.Message, prefix))
			if err == nil {
				waitTime := time.Duration(seconds) * time.Second
				if waitTime <= c.config.AutoRetry.MaxFloodWait {
					c.Logger.Info("Flood wait", "seconds", seconds)
					time.Sleep(waitTime)
					return true
				}
			}
		}
	}

	if errObj.Code != 400 {
		return false
	}

	if !*isMessageAttemptedLoad && errObj.Message == "Message not found" && c.config.AutoRetry.MessageNotFound {
		*isMessageAttemptedLoad = true
		var chatId, messageId int64
		if cId, ok := req["chat_id"].(float64); ok {
			chatId = int64(cId)
		}
		if mId, ok := req["message_id"].(float64); ok {
			messageId = int64(mId)
		}
		if chatId != 0 && messageId != 0 {
			c.Logger.Debug("Attempting to load message", "chat_id", chatId, "message_id", messageId)
			msg, loadErr := c.GetMessage(chatId, messageId)
			if loadErr == nil && msg != nil {
				c.Logger.Debug("Successfully loaded message, retrying request", "chat_id", chatId, "message_id", messageId)
				return true
			}

			c.Logger.Debug("Failed to load message", "chat_id", chatId, "message_id", messageId, "error", loadErr)
		}
	}

	if !*isChatAttemptedLoad && errObj.Message == "Chat not found" && c.config.AutoRetry.ChatNotFound {
		*isChatAttemptedLoad = true
		var chatId int64
		if cId, ok := req["chat_id"].(float64); ok {
			chatId = int64(cId)
		}
		if chatId != 0 {
			c.Logger.Debug("Attempting to load chat", "chat_id", chatId)
			chat, loadErr := c.GetChat(chatId)
			if loadErr == nil && chat != nil {
				c.Logger.Debug("Successfully loaded chat, retrying request", "chat_id", chatId)
				if replyToMap, ok := req["reply_to"].(map[string]interface{}); ok {
					if mId, ok := replyToMap["message_id"].(float64); ok {
						replyToMessageId := int64(mId)
						if replyToMessageId > 0 {
							_, _ = c.GetMessage(chatId, replyToMessageId)
						}
					}
				}

				return true
			}

			c.Logger.Debug("Failed to load chat", "chat_id", chatId, "error", loadErr)
		}
	}
	return false
}

// waitMessage waits for the message to be sent and returns the final message.
func (c *Client) waitMessage(msg *Message) (*Message, error) {
	if msg.SendingState != nil && msg.SendingState.GetType() == "messageSendingStatePending" {
		key := fmt.Sprintf("%d:%d", msg.ChatId, msg.Id)
		ch := make(chan TlObject, 1)
		c.pendingMessages.Store(key, ch)
		defer c.pendingMessages.Delete(key)

		select {
		case res := <-ch:
			if errObj, ok := res.(*Error); ok {
				return nil, errObj
			}
			if u, ok := res.(*UpdateMessageSendFailed); ok {
				return u.Message, u.Error
			}
			if finalMsg, ok := res.(*Message); ok {
				return finalMsg, nil
			}
			if u, ok := res.(*UpdateMessageSendSucceeded); ok {
				return u.Message, nil
			}
			return nil, fmt.Errorf("unexpected response type from waiter: %T", res)
		case <-time.After(30 * time.Second):
			return msg, nil
		}
	}

	return msg, nil
}

// waitMessages waits for all messages in the album to be sent and returns the final messages.
func (c *Client) waitMessages(msgs *Messages) (*Messages, error) {
	if msgs == nil || len(msgs.Messages) == 0 {
		return msgs, nil
	}

	totalCount := len(msgs.Messages)
	ch := make(chan TlObject, totalCount)
	errs := make([]error, totalCount)

	for i := range msgs.Messages {
		msg := &msgs.Messages[i]
		if msg.SendingState != nil && msg.SendingState.GetType() == "messageSendingStatePending" {
			key := fmt.Sprintf("%d:%d", msg.ChatId, msg.Id)
			c.pendingMessages.Store(key, ch)
		} else {
			ch <- msg
		}
	}

	defer func() {
		for i := range msgs.Messages {
			msg := &msgs.Messages[i]
			key := fmt.Sprintf("%d:%d", msg.ChatId, msg.Id)
			c.pendingMessages.Delete(key)
		}
	}()

	var receivedCount int
	timeout := time.After(30 * time.Second)

	for receivedCount < totalCount {
		select {
		case res := <-ch:
			if errObj, ok := res.(*Error); ok {
				errs[receivedCount] = errObj
			} else if u, ok := res.(*UpdateMessageSendFailed); ok {
				errs[receivedCount] = u.Error
				for i := range msgs.Messages {
					if msgs.Messages[i].Id == u.OldMessageId {
						msgs.Messages[i] = *u.Message
						break
					}
				}
			} else if finalMsg, ok := res.(*Message); ok {
				for i := range msgs.Messages {
					if msgs.Messages[i].Id == finalMsg.Id {
						msgs.Messages[i] = *finalMsg
						break
					}
				}
			} else if u, ok := res.(*UpdateMessageSendSucceeded); ok {
				for i := range msgs.Messages {
					if msgs.Messages[i].Id == u.OldMessageId {
						msgs.Messages[i] = *u.Message
						break
					}
				}
			}
			receivedCount++
		case <-timeout:
			return msgs, errors.Join(errs...)
		}
	}

	return msgs, errors.Join(errs...)
}

func toOptionValue(v interface{}) OptionValue {
	switch val := v.(type) {
	case bool:
		return &OptionValueBoolean{Value: val}
	case int:
		return &OptionValueInteger{Value: int64(val)}
	case int32:
		return &OptionValueInteger{Value: int64(val)}
	case int64:
		return &OptionValueInteger{Value: val}
	case string:
		return &OptionValueString{Value: val}
	case nil:
		return &OptionValueEmpty{}
	default:
		return nil
	}
}

// AddMessageHandler registers a handler for non-nil Message values carried by UpdateNewMessage updates.
func (c *Client) AddMessageHandler(hn MessageHandlerFunc, f ...Filter) Handle {
	return c.AddNewMessageHandler(func(c *Client, u *UpdateNewMessage) error {
		if u.Message == nil {
			return nil
		}
		return hn(c, u.Message)
	}, f...)
}

// AddCommandHandler registers a command handler for non-nil Message values carried by UpdateNewMessage updates.
func (c *Client) AddCommandHandler(command string, hn MessageHandlerFunc, f ...Filter) Handle {
	filters := append([]Filter{FilterCommand(command)}, f...)
	return c.AddMessageHandler(hn, filters...).SetPriority(10)
}

// OnCallback registers a handler for UpdateNewCallbackQuery updates.
func (c *Client) OnCallback(hn HandlerFunc[UpdateNewCallbackQuery], f ...Filter) Handle {
	return c.AddNewCallbackQueryHandler(hn, f...)
}

// OnCommand registers a command handler.
func (c *Client) OnCommand(command string, hn MessageHandlerFunc, f ...Filter) Handle {
	return c.AddCommandHandler(command, hn, f...)
}

// OnMessage registers a message handler.
func (c *Client) OnMessage(hn MessageHandlerFunc, f ...Filter) Handle {
	return c.AddMessageHandler(hn, f...)
}

// OnRaw registers a raw handler that accepts any update satisfying the filters.
func (c *Client) OnRaw(hn RawHandler, f ...Filter) Handle {
	return c.AddRawHandler(hn, f...)
}

// AddRawHandler registers a raw handler that accepts any update satisfying the filters.
func (c *Client) AddRawHandler(hn RawHandler, f ...Filter) Handle {
	h := &rawHandle{
		client:  c,
		handler: hn,
		filters: f,
	}

	c.addHandler(UpdateTypeRaw, h)

	return h
}

// RemoveHandler removes a registered update handler.
func (c *Client) RemoveHandler(h Handle) {
	if h == nil {
		return
	}
	tp := h.GetType()

	c.handleMu.Lock()
	defer c.handleMu.Unlock()

	handlers := c.handlers[tp]
	for i, handler := range handlers {
		if handler == h {
			c.handlers[tp] = append(handlers[:i], handlers[i+1:]...)
			return
		}
	}
}

func (c *Client) addHandler(tp UpdateType, h Handle) {
	c.handleMu.Lock()
	defer c.handleMu.Unlock()

	c.handlers[tp] = append(c.handlers[tp], h)
	c.sortHandlersLocked(tp)
}

func (c *Client) sortHandlers(tp UpdateType) {
	c.handleMu.Lock()
	defer c.handleMu.Unlock()
	c.sortHandlersLocked(tp)
}

func (c *Client) sortHandlersLocked(tp UpdateType) {
	sort.SliceStable(c.handlers[tp], func(i, j int) bool {
		if c.handlers[tp][i].GetGroup() != c.handlers[tp][j].GetGroup() {
			return c.handlers[tp][i].GetGroup() < c.handlers[tp][j].GetGroup()
		}
		return c.handlers[tp][i].GetPriority() > c.handlers[tp][j].GetPriority()
	})
}

func (c *Client) processUpdate(update TlObject) {
	go func() {
		tp := UpdateType(update.GetType())

		defer func() {
			if r := recover(); r != nil {
				if c.PanicHandler != nil {
					c.PanicHandler(c, update, r)
				}
			}
		}()

		// Waiters
		c.wMu.RLock()
		var matchedWaiters []*Waiter
		for _, w := range c.waiters {
			if w.Filter(c, update) {
				matchedWaiters = append(matchedWaiters, w)
			}
		}
		c.wMu.RUnlock()

		for _, w := range matchedWaiters {
			select {
			case w.C <- update:
			default:
			}
		}

		// Handlers
		c.handleMu.RLock()
		typeHandlers := c.handlers[tp]
		rawHandlers := c.handlers[UpdateTypeRaw]

		handlers := make([]Handle, 0, len(typeHandlers)+len(rawHandlers))
		i, j := 0, 0
		for i < len(typeHandlers) && j < len(rawHandlers) {
			h1, h2 := typeHandlers[i], rawHandlers[j]
			if h1.GetGroup() < h2.GetGroup() || (h1.GetGroup() == h2.GetGroup() && h1.GetPriority() >= h2.GetPriority()) {
				handlers = append(handlers, h1)
				i++
			} else {
				handlers = append(handlers, h2)
				j++
			}
		}
		handlers = append(handlers, typeHandlers[i:]...)
		handlers = append(handlers, rawHandlers[j:]...)
		c.handleMu.RUnlock()

		for i := 0; i < len(handlers); {
			currentGroup := handlers[i].GetGroup()
			groupHandled := false

			for ; i < len(handlers) && handlers[i].GetGroup() == currentGroup; i++ {
				if groupHandled {
					continue
				}

				h := handlers[i]
				if h.IsMatch(c, update) {
					groupHandled = true
					err := h.Execute(c, update)

					var action error
					if err != nil && !errors.Is(err, EndGroups) && !errors.Is(err, ContinueGroups) {
						action = c.ErrorHandler(c, update, err)
					} else {
						action = err
					}

					if errors.Is(action, EndGroups) {
						return
					}
					if errors.Is(action, ContinueGroups) {
						groupHandled = false
						continue
					}
				}
			}
		}
	}()
}

func (c *Client) WaitFor(filter func(*Client, TlObject) bool, timeout time.Duration) (TlObject, error) {
	ch := make(chan TlObject, 1)
	idNum := atomic.AddInt64(&c.waiterCount, 1)
	id := fmt.Sprintf("%d", idNum)

	c.wMu.Lock()
	c.waiters[id] = &Waiter{Filter: filter, C: ch}
	c.wMu.Unlock()

	defer func() {
		c.wMu.Lock()
		delete(c.waiters, id)
		c.wMu.Unlock()
	}()

	select {
	case u := <-ch:
		return u, nil
	case <-time.After(timeout):
		return nil, ConversationTimeout
	}
}

type Waiter struct {
	Filter func(*Client, TlObject) bool
	C      chan TlObject
}

func Bool(b bool) *bool {
	return &b
}
