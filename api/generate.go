// Package api contains the low-level client and data types generated from the
// Stirling PDF Processing API OpenAPI document by oapi-codegen.
//
// Most applications should use the ergonomic stirling package instead. The
// generated API is kept public as an escape hatch for new or self-hosted
// endpoints that are not yet covered by the high-level SDK.
package api

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.1 -config oapi-codegen.types.yaml openapi.json
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.1 -config oapi-codegen.client.yaml openapi.json
