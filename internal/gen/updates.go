package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

func generateUpdates(types []TLType) {
	var sb strings.Builder

	w := func(format string, args ...any) {
		fmt.Fprintf(&sb, format, args...)
	}

	w(header)
	w("package gotdbot\n\n")

	// UpdateType
	w("type UpdateType string\n\n")

	w("const (\n")
	for _, t := range types {
		if t.ResultType != "Update" {
			continue
		}

		w(
			"\t%s UpdateType = %q\n",
			"UpdateType"+strings.TrimPrefix(toCamelCase(t.Name), "Update"),
			t.Name,
		)
	}
	w(")\n\n")

	// On<Update>
	for _, t := range types {
		if t.ResultType != "Update" {
			continue
		}

		typ := toCamelCase(t.Name)
		constName := "UpdateType" + strings.TrimPrefix(typ, "Update")

		methodName := "Add" + strings.TrimPrefix(typ, "Update") + "Handler"
		w("// %s registers a handler for %s updates.\n", methodName, typ)
		w(`func (c *Client) %s(hn HandlerFunc[%s], f ...Filter) Handle {
	h := &handle[%s]{
		client:  c,
		handler: hn,
		filters: f,
		tp:      %s,
	}

`, methodName, typ, typ, constName)

		if typ == "UpdateNewMessage" {
			w("\th.applyDefaultFilters()\n")
		}

		w(`	c.addHandler(%s, h)

	return h
}

`, constName)

		onMethodName := "On" + strings.TrimPrefix(typ, "Update")
		w("// %s registers a handler for %s updates.\n", onMethodName, typ)
		w(`func (c *Client) %s(hn HandlerFunc[%s], f ...Filter) Handle {
	return c.%s(hn, f...)
}

`, onMethodName, typ, methodName)
	}

	// Generate ExtractChatID helper
	w("func ExtractChatID(u TlObject) int64 {\n")
	w("\tswitch upd := u.(type) {\n")
	for _, t := range types {
		if t.ResultType != "Update" {
			continue
		}
		typeName := toCamelCase(t.Name)

		hasMessage := false
		hasChatId := false
		hasChat := false
		hasUser := false
		hasSenderUserId := false
		hasUserId := false
		hasSenderId := false

		for _, p := range t.Params {
			switch p.Name {
			case "message":
				if p.Type == "message" {
					hasMessage = true
				}
			case "chat_id":
				hasChatId = true
			case "sender_user_id":
				hasSenderUserId = true
			case "user_id":
				hasUserId = true
			case "sender_id":
				hasSenderId = true
			default:
				if p.Type == "chat" {
					hasChat = true
				}
				if p.Type == "user" {
					hasUser = true
				}
			}
		}

		if hasMessage || hasChatId || hasChat || hasUser || hasSenderUserId || hasUserId || hasSenderId {
			w("\tcase *%s:\n", typeName)
			if hasChatId {
				w("\t\treturn upd.ChatId\n")
			} else if hasMessage {
				w("\t\tif upd.Message != nil { return upd.Message.ChatId }\n")
			} else if hasChat {
				for _, p := range t.Params {
					if p.Type == "chat" {
						w("\t\tif upd.%s != nil { return upd.%s.Id }\n", toCamelCase(p.Name), toCamelCase(p.Name))
						break
					}
				}
			} else if hasUser {
				for _, p := range t.Params {
					if p.Type == "user" {
						w("\t\tif upd.%s != nil { return upd.%s.Id }\n", toCamelCase(p.Name), toCamelCase(p.Name))
						break
					}
				}
			} else if hasSenderUserId {
				w("\t\treturn upd.SenderUserId\n")
			} else if hasUserId {
				w("\t\treturn upd.UserId\n")
			} else if hasSenderId {
				w("\t\tif up, ok := upd.SenderId.(*MessageSenderUser); ok { return up.UserId }\n")
				w("\t\tif up, ok := upd.SenderId.(*MessageSenderChat); ok { return up.ChatId }\n")
			}
		}
	}
	w("\t}\n\treturn 0\n}\n\n")

	// Generate ExtractSenderID helper
	w("func ExtractSenderID(u TlObject) int64 {\n")
	w("\tswitch upd := u.(type) {\n")
	for _, t := range types {
		if t.ResultType != "Update" {
			continue
		}
		typeName := toCamelCase(t.Name)

		hasSenderUserId := false
		hasSenderId := false
		hasMessage := false
		hasUserId := false

		for _, p := range t.Params {
			switch p.Name {
			case "sender_user_id":
				hasSenderUserId = true
			case "sender_id":
				hasSenderId = true
			case "message":
				if p.Type == "message" {
					hasMessage = true
				}
			case "user_id":
				hasUserId = true
			}
		}

		if hasSenderUserId || hasSenderId || hasMessage || hasUserId {
			w("\tcase *%s:\n", typeName)
			if hasSenderUserId {
				w("\t\treturn upd.SenderUserId\n")
			} else if hasSenderId {
				w("\t\tif up, ok := upd.SenderId.(*MessageSenderUser); ok { return up.UserId }\n")
				w("\t\tif up, ok := upd.SenderId.(*MessageSenderChat); ok { return up.ChatId }\n")
			} else if hasMessage {
				w("\t\tif upd.Message != nil { return upd.Message.SenderID() }\n")
			} else if hasUserId {
				w("\t\treturn upd.UserId\n")
			}
		}
	}
	w("\t}\n\treturn 0\n}\n\n")

	if err := os.WriteFile("gen_handlers.go", []byte(sb.String()), 0644); err != nil {
		log.Fatalf("Failed to write gen_updates.go: %v", err)
	}
}
