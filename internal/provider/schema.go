package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// schemaMode determines if we want a schema for:
// - reading single item - we need to provide "id" of the item to read, everything else is provided by the server,
// - reading list of items - we don't need to provide a thing, everything is provided by the server,
// - creating new item - we need to provide a bunch of obligatory attributes, the rest is provided by the server.
type schemaMode int

const (
	readSingle schemaMode = iota
	readList
	create
)

// presence says how an attribute behaves in the resource (create mode).
// In data sources (read modes) every attribute is computed.
type presence int

const (
	required         presence = iota // user must set it
	optional                         // user may set it; zero value otherwise
	optionalComputed                 // user may set it; the server fills it otherwise
	computedOnly                     // set by the server only
)

// field adapts s to mode: in create mode it applies p; in read modes it strips
// everything that only makes sense for user input and marks it computed.
func field(mode schemaMode, p presence, s *schema.Schema) *schema.Schema {
	if mode != create {
		s.Required, s.Optional, s.Computed = false, false, true
		s.Default, s.Deprecated, s.ValidateDiagFunc, s.MaxItems = nil, "", nil, 0
		s.ExactlyOneOf, s.ConflictsWith = nil, nil
		return s
	}
	switch p {
	case required:
		s.Required = true
	case optional:
		s.Optional = true
	case optionalComputed:
		s.Optional, s.Computed = true, true
	case computedOnly:
		s.Computed = true
		s.Default, s.ValidateDiagFunc, s.MaxItems = nil, nil, 0
	}
	return s
}

// block is a nested single-object block (Terraform SDKv2 models it as a list of one).
func block(mode schemaMode, p presence, description string, attrs map[string]*schema.Schema) *schema.Schema {
	return field(mode, p, &schema.Schema{
		Type:        schema.TypeList,
		MaxItems:    1,
		Description: description,
		Elem:        &schema.Resource{Schema: attrs},
	})
}
