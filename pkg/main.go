package pkg

import (
	"encoding/json"
	"fmt"
)

// IDTypeMapping represents how the GraphQL ID type should be mapped in JSON Schema
type IDTypeMapping string

const (
	IDTypeString      IDTypeMapping = "string"
	IDTypeNumber      IDTypeMapping = "number"
	IDTypeBoth        IDTypeMapping = "both"
	IDTypeDefaultMode               = IDTypeString
)

// Options contains configuration options for the conversion process
type Options struct {
	IgnoreInternals    bool          `json:"ignoreInternals"`
	NullableArrayItems bool          `json:"nullableArrayItems"`
	IDTypeMapping      IDTypeMapping `json:"idTypeMapping"`
}

// DefaultOptions returns the default conversion options
func DefaultOptions() Options {
	return Options{
		IgnoreInternals:    true,
		NullableArrayItems: false,
		IDTypeMapping:      IDTypeDefaultMode,
	}
}

// JSONSchema6 represents a JSON Schema Draft 6 schema
type JSONSchema6 struct {
	Schema      string                  `json:"$schema"`
	Type        interface{}             `json:"type,omitempty"`
	Properties  map[string]*JSONSchema6 `json:"properties,omitempty"`
	Items       *JSONSchema6            `json:"items,omitempty"`
	Ref         string                  `json:"$ref,omitempty"`
	Required    []string                `json:"required,omitempty"`
	Definitions map[string]*JSONSchema6 `json:"definitions,omitempty"`
	AnyOf       []*JSONSchema6          `json:"anyOf,omitempty"`
	OneOf       []*JSONSchema6          `json:"oneOf,omitempty"`
	Title       string                  `json:"title,omitempty"`
	Description string                  `json:"description,omitempty"`
	Default     interface{}             `json:"default,omitempty"`
	Enum        []string                `json:"enum,omitempty"`
}

// IntrospectionQuery represents the root of a GraphQL introspection query result
type IntrospectionQuery struct {
	Schema IntrospectionSchema `json:"__schema"`
}

// IntrospectionSchema represents the schema information from an introspection query
type IntrospectionSchema struct {
	QueryType    *TypeRef            `json:"queryType"`
	MutationType *TypeRef            `json:"mutationType"`
	Types        []IntrospectionType `json:"types"`
}

// TypeRef represents a reference to a type
type TypeRef struct {
	Name string `json:"name"`
}

// IntrospectionType represents a type in the GraphQL schema
type IntrospectionType struct {
	Kind          string               `json:"kind"`
	Name          string               `json:"name"`
	Description   string               `json:"description"`
	Fields        []IntrospectionField `json:"fields"`
	InputFields   []IntrospectionInput `json:"inputFields"`
	Interfaces    []TypeRef            `json:"interfaces"`
	EnumValues    []IntrospectionEnum  `json:"enumValues"`
	PossibleTypes []IntrospectionType  `json:"possibleTypes"`
}

// IntrospectionField represents a field in a GraphQL type
type IntrospectionField struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Args        []IntrospectionArg   `json:"args"`
	Type        IntrospectionTypeRef `json:"type"`
}

// IntrospectionInput represents an input field in a GraphQL type
type IntrospectionInput struct {
	Name         string               `json:"name"`
	Description  string               `json:"description"`
	Type         IntrospectionTypeRef `json:"type"`
	DefaultValue *string              `json:"defaultValue"`
}

// IntrospectionArg represents an argument to a field
type IntrospectionArg struct {
	Name         string               `json:"name"`
	Description  string               `json:"description"`
	Type         IntrospectionTypeRef `json:"type"`
	DefaultValue *string              `json:"defaultValue"`
}

// IntrospectionEnum represents an enum value in a GraphQL enum type
type IntrospectionEnum struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// IntrospectionTypeRef represents a type reference in the schema
type IntrospectionTypeRef struct {
	Kind   string                `json:"kind"`
	Name   *string               `json:"name"`
	OfType *IntrospectionTypeRef `json:"ofType"`
}

// FromIntrospectionQuery converts a GraphQL introspection query result to a JSON Schema
func FromIntrospectionQuery(introspection IntrospectionQuery, opts *Options) (*JSONSchema6, error) {
	if opts == nil {
		defaultOpts := DefaultOptions()
		opts = &defaultOpts
	}

	schema := &JSONSchema6{
		Schema:      "http://json-schema.org/draft-06/schema#",
		Properties:  make(map[string]*JSONSchema6),
		Definitions: make(map[string]*JSONSchema6),
	}

	if introspection.Schema.QueryType != nil && introspection.Schema.Types != nil {
		queryType := findType(introspection.Schema.Types, introspection.Schema.QueryType.Name)
		if queryType != nil {
			schema.Properties["Query"] = processType(*queryType, opts)
		}
	}

	if introspection.Schema.MutationType != nil && introspection.Schema.Types != nil {
		mutationType := findType(introspection.Schema.Types, introspection.Schema.MutationType.Name)
		if mutationType != nil {
			schema.Properties["Mutation"] = processType(*mutationType, opts)
		}
	}

	if introspection.Schema.Types != nil {
		filteredTypes := filterTypes(introspection.Schema.Types, opts.IgnoreInternals)
		for _, t := range filteredTypes {
			if !isRootType(t.Name) {
				schema.Definitions[t.Name] = processType(t, opts)
			}
		}
	}

	return schema, nil
}

// Helper functions

func isRootType(name string) bool {
	return name == "Query" || name == "Mutation"
}

func findType(types []IntrospectionType, name string) *IntrospectionType {
	for _, t := range types {
		if t.Name == name {
			return &t
		}
	}
	return nil
}

func filterTypes(types []IntrospectionType, ignoreInternals bool) []IntrospectionType {
	if !ignoreInternals {
		return types
	}

	filtered := make([]IntrospectionType, 0)
	for _, t := range types {
		if len(t.Name) < 2 || t.Name[:2] != "__" {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func processType(t IntrospectionType, opts *Options) *JSONSchema6 {
	schema := &JSONSchema6{
		Type:        "object",
		Properties:  make(map[string]*JSONSchema6),
		Description: t.Description,
	}

	switch t.Kind {
	case "OBJECT", "INTERFACE":
		required := make([]string, 0)
		if t.Fields != nil {
			for _, field := range t.Fields {
				schema.Properties[field.Name] = processField(field, opts)
				if isRequired(field.Type) {
					required = append(required, field.Name)
				}
			}
		}
		if len(required) > 0 {
			schema.Required = required
		}

	case "INPUT_OBJECT":
		required := make([]string, 0)
		if t.InputFields != nil {
			for _, field := range t.InputFields {
				schema.Properties[field.Name] = processInputValue(field, opts)
				if isRequired(field.Type) {
					required = append(required, field.Name)
				}
			}
		}
		if len(required) > 0 {
			schema.Required = required
		}

	case "ENUM":
		schema.Type = "string"
		anyOf := make([]*JSONSchema6, 0)
		if t.EnumValues != nil {
			for _, enumValue := range t.EnumValues {
				anyOf = append(anyOf, &JSONSchema6{
					Enum:        []string{enumValue.Name},
					Title:       enumValue.Description,
					Description: enumValue.Description,
				})
			}
		}
		schema.AnyOf = anyOf

	case "UNION":
		delete(schema.Properties, "type")
		oneOf := make([]*JSONSchema6, 0)
		if t.PossibleTypes != nil {
			for _, possibleType := range t.PossibleTypes {
				oneOf = append(oneOf, &JSONSchema6{
					Ref: fmt.Sprintf("#/definitions/%s", possibleType.Name),
				})
			}
		}
		schema.OneOf = oneOf
	}

	return schema
}

func processField(field IntrospectionField, opts *Options) *JSONSchema6 {
	schema := &JSONSchema6{
		Type:        "object",
		Properties:  make(map[string]*JSONSchema6),
		Description: field.Description,
	}

	// Process return type
	schema.Properties["return"] = processTypeRef(field.Type, opts)

	// Process arguments
	args := &JSONSchema6{
		Type:       "object",
		Properties: make(map[string]*JSONSchema6),
	}

	required := make([]string, 0)
	if field.Args != nil {
		for _, arg := range field.Args {
			args.Properties[arg.Name] = processArg(arg, opts)
			if isRequired(arg.Type) {
				required = append(required, arg.Name)
			}
		}
	}

	if len(required) > 0 {
		args.Required = required
	}

	schema.Properties["arguments"] = args

	return schema
}

func processInputValue(input IntrospectionInput, opts *Options) *JSONSchema6 {
	schema := processTypeRef(input.Type, opts)
	schema.Description = input.Description

	if input.DefaultValue != nil {
		var defaultValue interface{}
		if err := json.Unmarshal([]byte(*input.DefaultValue), &defaultValue); err == nil {
			schema.Default = defaultValue
		}
	}

	return schema
}

func processArg(arg IntrospectionArg, opts *Options) *JSONSchema6 {
	schema := processTypeRef(arg.Type, opts)
	schema.Description = arg.Description

	if arg.DefaultValue != nil {
		var defaultValue interface{}
		if err := json.Unmarshal([]byte(*arg.DefaultValue), &defaultValue); err == nil {
			schema.Default = defaultValue
		}
	}

	return schema
}

func processTypeRef(typeRef IntrospectionTypeRef, opts *Options) *JSONSchema6 {
	switch typeRef.Kind {
	case "NON_NULL":
		if typeRef.OfType != nil {
			return processTypeRef(*typeRef.OfType, opts)
		}
		return &JSONSchema6{}
	case "LIST":
		if typeRef.OfType != nil {
			items := processTypeRef(*typeRef.OfType, opts)
			schema := &JSONSchema6{
				Type:  "array",
				Items: items,
			}

			if opts.NullableArrayItems && !isRequired(*typeRef.OfType) {
				schema.Items = &JSONSchema6{
					AnyOf: []*JSONSchema6{
						items,
						{Type: "null"},
					},
				}
			}

			return schema
		}
		return &JSONSchema6{Type: "array"}
	case "SCALAR":
		if typeRef.Name != nil {
			return processScalar(*typeRef.Name, opts.IDTypeMapping)
		}
		return &JSONSchema6{}
	default:
		if typeRef.Name != nil {
			return &JSONSchema6{
				Ref: fmt.Sprintf("#/definitions/%s", *typeRef.Name),
			}
		}
		return &JSONSchema6{}
	}
}

func processScalar(name string, idMapping IDTypeMapping) *JSONSchema6 {
	schema := &JSONSchema6{
		Title: name,
	}

	switch name {
	case "ID":
		schema.Description = "The `ID` scalar type represents a unique identifier, often used to refetch an object or as key for a cache. The ID type appears in a JSON response as a String; however, it is not intended to be human-readable. When expected as an input type, any string (such as `\"4\"`) or integer (such as `4`) input value will be accepted as an ID."

		switch idMapping {
		case IDTypeNumber:
			schema.Type = "number"
		case IDTypeBoth:
			schema.Type = []string{"string", "number"}
		default:
			schema.Type = "string"
		}

	case "String":
		schema.Type = "string"
		schema.Description = "The `String` scalar type represents textual data, represented as UTF-8 character sequences. The String type is most often used by GraphQL to represent free-form human-readable text."

	case "Int", "Float":
		schema.Type = "number"

	case "Boolean":
		schema.Type = "boolean"
		schema.Description = "The `Boolean` scalar type represents `true` or `false`."
	}

	return schema
}

func isRequired(typeRef IntrospectionTypeRef) bool {
	return typeRef.Kind == "NON_NULL"
}

// IsValidIDTypeMapping checks if the provided IDTypeMapping is valid
func IsValidIDTypeMapping(mapping IDTypeMapping) bool {
	validMappings := []IDTypeMapping{"string", "number", "both"}
	for _, valid := range validMappings {
		if mapping == valid {
			return true
		}
	}

	return false
}
