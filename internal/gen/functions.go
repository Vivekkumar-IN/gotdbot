package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

func generateFunctions(functions []TLType, classes map[string]*TLClass) {
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("package gotdbot\n\n")
	sb.WriteString("import \"encoding/json\"\n\n")

	for _, t := range functions {
		structName := toCamelCase(t.Name)
		fmt.Fprintf(&sb, "// %s %s\n", structName, formatDesc(t.Description))
		fmt.Fprintf(&sb, "type %s struct {\n", structName)

		for _, p := range t.Params {
			goType := toGoType(p.Type, classes)
			fieldName := toCamelCase(p.Name)
			jsonTag := fmt.Sprintf("`json:\"%s\"`", p.Name)
			if p.Type == "int64" {
				jsonTag = fmt.Sprintf("`json:\"%s,string\"`", p.Name)
			}
			if p.IsOptional {
				if p.Type == "int64" {
					jsonTag = fmt.Sprintf("`json:\"%s,string,omitempty\"`", p.Name)
				} else {
					jsonTag = fmt.Sprintf("`json:\"%s,omitempty\"`", p.Name)
				}
			}
			fmt.Fprintf(&sb, "\t// %s\n", formatDesc(p.Description))
			fmt.Fprintf(&sb, "\t%s %s %s\n", fieldName, goType, jsonTag)
		}
		sb.WriteString("}\n\n")

		fmt.Fprintf(&sb, "func (t %s) GetType() string {\n", structName)
		fmt.Fprintf(&sb, "\treturn \"%s\"\n", t.Name)
		sb.WriteString("}\n\n")

		// MarshalJSON (only @type)
		fmt.Fprintf(&sb, "func (t %s) MarshalJSON() ([]byte, error) {\n", structName)
		fmt.Fprintf(&sb, "\ttype Alias %s\n", structName)
		sb.WriteString("\treturn json.Marshal(&struct {\n")
		sb.WriteString("\t\tTypeStr string `json:\"@type\"`\n")
		sb.WriteString("\t\t*Alias\n")
		sb.WriteString("\t}{\n")
		fmt.Fprintf(&sb, "\t\tTypeStr: \"%s\",\n", t.Name)
		sb.WriteString("\t\tAlias:   (*Alias)(&t),\n")
		sb.WriteString("\t})\n")
		sb.WriteString("}\n\n")
	}

	if err := os.WriteFile("gen_functions.go", []byte(sb.String()), 0644); err != nil {
		log.Fatal(err)
	}
}
