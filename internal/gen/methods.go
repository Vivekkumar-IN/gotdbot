package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

func generateMethods(functions []TLType, classes map[string]*TLClass) {
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("package gotdbot\n\n")

	for _, fn := range functions {
		methodName := toCamelCase(fn.Name)

		if methodName == "Close" {
			continue
		}
		structName := toCamelCase(fn.Name)

		hasOptional := false
		for _, p := range fn.Params {
			if p.IsOptional || p.Type == "Bool" {
				hasOptional = true
				break
			}
		}

		optsStructName := methodName + "Opts"

		// Generate Opts struct (inline, before method)
		if hasOptional {
			fmt.Fprintf(&sb, "// %s contains optional parameters for %s\n", optsStructName, methodName)
			fmt.Fprintf(&sb, "type %s struct {\n", optsStructName)
			for _, p := range fn.Params {
				if p.IsOptional || p.Type == "Bool" {
					fmt.Fprintf(&sb, "\t// %s\n", formatDesc(p.Description))
					goType := toGoType(p.Type, classes)
					fieldName := toCamelCase(p.Name)
					fmt.Fprintf(&sb, "\t%s %s\n", fieldName, goType)
				}
			}
			sb.WriteString("}\n\n")
		}

		isOk := fn.ResultType == "ok" || fn.ResultType == "Ok"
		resultType := toCamelCase(fn.ResultType)
		if isOk {
			resultType = "Ok"
		}

		retTypeStr := "*" + resultType
		if _, ok := classes[fn.ResultType]; ok {
			retTypeStr = toCamelCase(fn.ResultType)
		}

		fmt.Fprintf(&sb, "// %s %s\n", methodName, formatDesc(fn.Description))
		fmt.Fprintf(&sb, "func (c *Client) %s(", methodName)

		// Args
		var args []string
		for _, p := range fn.Params {
			if p.IsOptional || p.Type == "Bool" {
				continue
			}
			goType := toGoType(p.Type, classes)
			fieldName := toCamelCase(p.Name)
			argName := strings.ToLower(fieldName[:1]) + fieldName[1:]
			if argName == "type" {
				argName = "typeField"
			}
			if argName == "func" {
				argName = "funcArg"
			}
			args = append(args, fmt.Sprintf("%s %s", argName, goType))
		}

		if hasOptional {
			args = append(args, fmt.Sprintf("opts ...*%s", optsStructName))
		}

		fmt.Fprintf(&sb, "%s", strings.Join(args, ", "))

		if isOk {
			fmt.Fprintf(&sb, ") error {\n")
		} else {
			fmt.Fprintf(&sb, ") (%s, error) {\n", retTypeStr)
		}

		fmt.Fprintf(&sb, "\treq := &%s{\n", structName)
		for _, p := range fn.Params {
			if p.IsOptional || p.Type == "Bool" {
				continue
			}
			fieldName := toCamelCase(p.Name)
			argName := strings.ToLower(fieldName[:1]) + fieldName[1:]
			if argName == "type" {
				argName = "typeField"
			}
			if argName == "func" {
				argName = "funcArg"
			}
			fmt.Fprintf(&sb, "\t\t%s: %s,\n", fieldName, argName)
		}
		sb.WriteString("\t}\n")

		if hasOptional {
			fmt.Fprintf(&sb, "\topt := getVariadic(opts, &%s{})\n", optsStructName)
			for _, p := range fn.Params {
				if !p.IsOptional && p.Type != "Bool" {
					continue
				}
				fieldName := toCamelCase(p.Name)
				fmt.Fprintf(&sb, "\treq.%s = opt.%s\n", fieldName, fieldName)
			}
		}

		if isOk {
			sb.WriteString("\t_, err := c.Send(req)\n")
			sb.WriteString("\treturn err\n")
		} else {
			sb.WriteString("\tresp, err := c.Send(req)\n")
			sb.WriteString("\tif err != nil {\n\t\treturn nil, err\n\t}\n")

			if methodName == "SendMessage" {
				sb.WriteString("\treturn c.waitMessage(resp.(*Message))\n")
			} else if methodName == "SendMessageAlbum" {
				sb.WriteString("\treturn c.waitMessages(resp.(*Messages))\n")
			} else {
				fmt.Fprintf(&sb, "\treturn resp.(%s), nil\n", retTypeStr)
			}
		}
		sb.WriteString("}\n\n")
	}

	if err := os.WriteFile("gen_methods.go", []byte(sb.String()), 0644); err != nil {
		log.Fatal(err)
	}
}
