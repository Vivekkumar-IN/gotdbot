package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
)

func generateOptions(options map[string]*OptionDef) {
	// Filter writable options and sort them
	var writableOptions []string
	for name, def := range options {
		if def.Writable {
			writableOptions = append(writableOptions, name)
		}
	}
	sort.Strings(writableOptions)

	contentBytes, err := os.ReadFile("client_opts.go")
	if err != nil {
		log.Fatal(err)
	}
	content := string(contentBytes)

	structRegex := regexp.MustCompile(`(?s)(// TDLibOptions.*?\n)?type TDLibOptions struct\s*\{`)
	forEachRegex := regexp.MustCompile(`(?s)(// forEachSet.*?\n)?func\s*\(o\s*\*TDLibOptions\)\s*forEachSet\s*\(fn func\(name string, value interface\{\}\)\)\s*\{`)

	var structSb, forEachSb strings.Builder
	structSb.WriteString("// TDLibOptions contains TDLib options that can be set\ntype TDLibOptions struct {\n")
	for _, name := range writableOptions {
		def := options[name]
		fieldName := toCamelCase(name)
		var goType string
		switch def.Type {
		case "Bool":
			goType = "bool"
		case "int64":
			goType = "int64"
		case "string":
			goType = "string"
		default:
			goType = "interface{}"
		}
		fmt.Fprintf(&structSb, "\t// %s\n", formatDesc(def.Description))
		fmt.Fprintf(&structSb, "\t%s %s `json:\"%s,omitempty\"`\n", fieldName, goType, name)
	}
	structSb.WriteString("}")

	forEachSb.WriteString("// forEachSet calls fn for each non-default TDLib option.\nfunc (o *TDLibOptions) forEachSet(fn func(name string, value interface{})) {\n\tif o == nil || fn == nil { return }\n")
	for _, name := range writableOptions {
		fieldName := toCamelCase(name)
		switch options[name].Type {
		case "Bool":
			fmt.Fprintf(&forEachSb, "\tif o.%s { fn(\"%s\", o.%s) }\n", fieldName, name, fieldName)
		case "int64":
			fmt.Fprintf(&forEachSb, "\tif o.%s != 0 { fn(\"%s\", o.%s) }\n", fieldName, name, fieldName)
		case "string":
			fmt.Fprintf(&forEachSb, "\tif o.%s != \"\" { fn(\"%s\", o.%s) }\n", fieldName, name, fieldName)
		default:
			fmt.Fprintf(&forEachSb, "\tif o.%s != nil { fn(\"%s\", o.%s) }\n", fieldName, name, fieldName)
		}
	}
	forEachSb.WriteString("}")

	newContent := content
	if loc := structRegex.FindStringIndex(newContent); loc != nil {
		end := findBalancedBrace(newContent, loc[1]-1)
		if end != -1 {
			for end+1 < len(newContent) && (newContent[end+1] == '\n' || newContent[end+1] == '\r') {
				end++
			}
			newContent = newContent[:loc[0]] + structSb.String() + "\n\n" + newContent[end+1:]
		}
	}

	if loc := forEachRegex.FindStringIndex(newContent); loc != nil {
		end := findBalancedBrace(newContent, loc[1]-1)
		if end != -1 {
			for end+1 < len(newContent) && (newContent[end+1] == '\n' || newContent[end+1] == '\r') {
				end++
			}
			newContent = newContent[:loc[0]] + forEachSb.String() + "\n\n" + newContent[end+1:]
		}
	} else if structLoc := structRegex.FindStringIndex(newContent); structLoc != nil {
		structEnd := findBalancedBrace(newContent, structLoc[1]-1)
		if structEnd != -1 {
			insertAt := structEnd + 1
			newContent = newContent[:insertAt] + "\n\n" + forEachSb.String() + newContent[insertAt:]
		}
	}

	if err := os.WriteFile("client_opts.go", []byte(newContent), 0644); err != nil {
		log.Fatal(err)
	}
}
