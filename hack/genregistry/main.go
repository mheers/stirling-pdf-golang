// Command genregistry turns the vendored OpenAPI document into the static
// operation registry and the generated operations reference.
//
// Usage:
//
//	go run ./hack/genregistry -spec api/openapi.json -out internal/ops/registry.gen.go -docs docs/operations.md
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type spec struct {
	Paths      map[string]map[string]operation `json:"paths"`
	Components struct {
		Schemas map[string]schema `json:"schemas"`
	} `json:"components"`
}

type operation struct {
	Tags        []string `json:"tags"`
	Summary     string   `json:"summary"`
	Description string   `json:"description"`
	OperationID string   `json:"operationId"`
	RequestBody *struct {
		Content map[string]struct {
			Schema schema `json:"schema"`
		} `json:"content"`
	} `json:"requestBody"`
	Responses map[string]struct {
		Content map[string]json.RawMessage `json:"content"`
	} `json:"responses"`
}

type schema struct {
	Ref         string              `json:"$ref"`
	Type        string              `json:"type"`
	Format      string              `json:"format"`
	Description string              `json:"description"`
	Required    []string            `json:"required"`
	Properties  map[string]property `json:"properties"`
	Items       *property           `json:"items"`
	Enum        []any               `json:"enum"`
	Default     any                 `json:"default"`
}

type property struct {
	Ref         string              `json:"$ref"`
	Type        string              `json:"type"`
	Format      string              `json:"format"`
	Description string              `json:"description"`
	Enum        []any               `json:"enum"`
	Default     any                 `json:"default"`
	Items       *property           `json:"items"`
	Properties  map[string]property `json:"properties"`
}

type field struct {
	Name        string
	Kind        string
	Required    bool
	Description string
	Enum        []string
	Default     string
	HasDefault  bool
}

type opDef struct {
	Group       string
	Name        string
	Path        string
	OperationID string
	Summary     string
	Description string
	Response    string
	Filter      bool
	Fields      []field
}

var groupOrder = []string{"General", "Convert", "Security", "Misc", "Analysis", "Filter", "Pipeline"}

func main() {
	specPath := flag.String("spec", "api/openapi.json", "OpenAPI document")
	outPath := flag.String("out", "internal/ops/registry.gen.go", "generated Go file")
	docsPath := flag.String("docs", "docs/operations.md", "generated operations reference")
	flag.Parse()

	raw, err := os.ReadFile(*specPath)
	if err != nil {
		log.Fatal(err)
	}
	var document spec
	if err := json.Unmarshal(raw, &document); err != nil {
		log.Fatal(err)
	}

	operations := buildOperations(document)
	if err := writeGo(*outPath, operations); err != nil {
		log.Fatal(err)
	}
	if err := writeDocs(*docsPath, operations); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("generated %d operations\n", len(operations))
}

func buildOperations(document spec) []opDef {
	var operations []opDef
	for apiPath, methods := range document.Paths {
		op, ok := methods["post"]
		if !ok {
			continue
		}
		group := "Misc"
		if len(op.Tags) > 0 {
			group = op.Tags[0]
		}
		defined := opDef{
			Group:       group,
			Name:        commandName(apiPath, group),
			Path:        apiPath,
			OperationID: op.OperationID,
			Summary:     op.Summary,
			Description: op.Description,
			Response:    responseKind(op.Responses["200"].Content),
			Filter:      group == "Filter",
		}
		if apiPath == "/api/v1/misc/auto-rename" {
			defined.Response = "text"
		}
		if op.RequestBody != nil {
			content, ok := op.RequestBody.Content["multipart/form-data"]
			if ok {
				body := content.Schema
				if body.Ref != "" {
					body = resolve(document, body.Ref)
				}
				defined.Fields = buildFields(document, body)
			}
		}
		operations = append(operations, defined)
	}
	sort.Slice(operations, func(i, j int) bool {
		if operations[i].Group != operations[j].Group {
			return groupRank(operations[i].Group) < groupRank(operations[j].Group)
		}
		return operations[i].Name < operations[j].Name
	})
	return operations
}

func groupRank(group string) int {
	for i, candidate := range groupOrder {
		if candidate == group {
			return i
		}
	}
	return len(groupOrder)
}

func commandName(apiPath, group string) string {
	trimmed := strings.TrimPrefix(apiPath, "/api/v1/")
	parts := strings.Split(trimmed, "/")
	if len(parts) > 0 && strings.EqualFold(parts[0], group) {
		parts = parts[1:]
	}
	for i, part := range parts {
		parts[i] = kebab(part)
	}
	return strings.Join(parts, "-")
}

func buildFields(document spec, body schema) []field {
	required := map[string]bool{}
	for _, name := range body.Required {
		required[name] = true
	}
	names := make([]string, 0, len(body.Properties))
	for name := range body.Properties {
		names = append(names, name)
	}
	sort.Strings(names)
	fields := make([]field, 0, len(names))
	for _, name := range names {
		prop := body.Properties[name]
		defined := field{
			Name:        name,
			Kind:        fieldKind(document, prop),
			Required:    required[name],
			Description: collapse(prop.Description),
		}
		if prop.Default != nil {
			defined.HasDefault = true
			defined.Default = fmt.Sprint(prop.Default)
		}
		for _, value := range prop.Enum {
			defined.Enum = append(defined.Enum, fmt.Sprint(value))
		}
		fields = append(fields, defined)
	}
	return fields
}

func fieldKind(document spec, prop property) string {
	if prop.Type == "string" && prop.Format == "binary" {
		return "File"
	}
	switch prop.Type {
	case "integer":
		return "Integer"
	case "number":
		return "Number"
	case "boolean":
		return "Boolean"
	case "object":
		return "StringMap"
	case "array":
		if prop.Items == nil {
			return "StringList"
		}
		item := resolveProperty(document, *prop.Items)
		if item.Format == "binary" {
			return "FileList"
		}
		switch item.Type {
		case "object":
			return "ObjectList"
		case "integer":
			return "IntegerList"
		case "number":
			return "NumberList"
		}
		return "StringList"
	}
	return "String"
}

func resolveProperty(document spec, prop property) property {
	if prop.Ref == "" {
		return prop
	}
	resolved := resolve(document, prop.Ref)
	return property{
		Type:        resolved.Type,
		Format:      resolved.Format,
		Description: resolved.Description,
		Enum:        resolved.Enum,
		Default:     resolved.Default,
		Items:       resolved.Items,
		Properties:  resolved.Properties,
	}
}

func responseKind(content map[string]json.RawMessage) string {
	if len(content) == 0 {
		return "binary"
	}
	text := false
	for contentType := range content {
		switch {
		case contentType == "application/json":
			// JSON wins if it is the only response type.
		case strings.HasPrefix(contentType, "text/"), contentType == "application/xml":
			text = true
		default:
			return "binary"
		}
	}
	if text && len(content) == 1 {
		return "text"
	}
	if _, ok := content["application/json"]; ok && len(content) == 1 {
		return "json"
	}
	return "binary"
}

func resolve(document spec, ref string) schema {
	name := strings.TrimPrefix(ref, "#/components/schemas/")
	return document.Components.Schemas[name]
}

func kebab(s string) string {
	var b strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				previous := runes[i-1]
				var next rune
				if i+1 < len(runes) {
					next = runes[i+1]
				}
				if unicode.IsLower(previous) || unicode.IsDigit(previous) || (unicode.IsUpper(previous) && unicode.IsLower(next)) {
					b.WriteByte('-')
				}
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		if r == '_' || r == ' ' {
			b.WriteByte('-')
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

func collapse(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func flagName(fieldName string) string {
	switch fieldName {
	case "fileInput":
		return "file"
	case "fileId":
		return "file-id"
	case "imageFile":
		return "image"
	case "overlayFiles":
		return "overlay-file"
	case "stampImage":
		return "stamp-image"
	case "attachments":
		return "attachment"
	}
	name := kebab(fieldName)
	return strings.ReplaceAll(name, "java-script", "javascript")
}

func writeGo(filename string, operations []opDef) error {
	var b bytes.Buffer
	b.WriteString(`// Code generated by hack/genregistry. DO NOT EDIT.

package ops

// FieldKind describes how a form field is exposed as a CLI flag.
type FieldKind string

// Supported field kinds.
const (
	StringKind     FieldKind = "string"
	IntegerKind    FieldKind = "integer"
	NumberKind     FieldKind = "number"
	BooleanKind    FieldKind = "boolean"
	StringListKind FieldKind = "string-list"
	IntegerListKind FieldKind = "integer-list"
	NumberListKind FieldKind = "number-list"
	FileKind       FieldKind = "file"
	FileListKind   FieldKind = "file-list"
	StringMapKind  FieldKind = "string-map"
	ObjectListKind FieldKind = "object-list"
)

// Field is one multipart form field of an operation.
type Field struct {
	Name        string
	Flag        string
	Kind        FieldKind
	Required    bool
	Description string
	Enum        []string
	Default     string
	HasDefault  bool
}

// Operation describes an endpoint from the Processing API.
type Operation struct {
	Group       string
	Name        string
	Path        string
	OperationID string
	Summary     string
	Description string
	Response    string
	Filter      bool
	Fields      []Field
}

// All contains every operation, ordered by group and command name.
var All = []Operation{
`)
	for _, op := range operations {
		fmt.Fprintf(&b, "\t{\n\t\tGroup: %s, Name: %s, Path: %s,\n\t\tOperationID: %s, Summary: %s,\n\t\tDescription: %s,\n\t\tResponse: %s, Filter: %t,\n\t\tFields: []Field{\n",
			strconv.Quote(op.Group), strconv.Quote(op.Name), strconv.Quote(op.Path),
			strconv.Quote(op.OperationID), strconv.Quote(op.Summary),
			strconv.Quote(op.Description), strconv.Quote(op.Response), op.Filter)
		for _, f := range op.Fields {
			fmt.Fprintf(&b, "\t\t\t{Name: %s, Flag: %s, Kind: %sKind, Required: %t, Description: %s",
				strconv.Quote(f.Name), strconv.Quote(flagName(f.Name)), f.Kind, f.Required, strconv.Quote(f.Description))
			if len(f.Enum) > 0 {
				b.WriteString(", Enum: []string{")
				for i, value := range f.Enum {
					if i > 0 {
						b.WriteString(", ")
					}
					b.WriteString(strconv.Quote(value))
				}
				b.WriteString("}")
			}
			if f.HasDefault {
				fmt.Fprintf(&b, ", Default: %s, HasDefault: true", strconv.Quote(f.Default))
			}
			b.WriteString("},\n")
		}
		b.WriteString("\t\t},\n\t},\n")
	}
	b.WriteString("}\n")

	formatted, err := format.Source(b.Bytes())
	if err != nil {
		return fmt.Errorf("format generated Go: %w\n%s", err, b.Bytes())
	}
	return os.WriteFile(filename, formatted, 0o644)
}

func writeDocs(filename string, operations []opDef) error {
	var b bytes.Buffer
	b.WriteString("# Operations\n\n")
	b.WriteString("Generated from `api/openapi.json` by `go generate ./...`. Every processing operation is available\n")
	b.WriteString("as `stirling-pdf <group> <command>` and through the matching SDK method.\n")
	current := ""
	for _, op := range operations {
		if op.Group != current {
			current = op.Group
			fmt.Fprintf(&b, "\n## %s\n\n", op.Group)
			b.WriteString("| Command | Path | Summary |\n|---|---|---|\n")
		}
		fmt.Fprintf(&b, "| `%s %s` | `%s` | %s |\n", strings.ToLower(op.Group), op.Name, op.Path, escapeTable(op.Summary))
	}
	if err := os.MkdirAll(path.Dir(filename), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filename, b.Bytes(), 0o644)
}

func escapeTable(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	return strings.ReplaceAll(s, "\n", " ")
}
