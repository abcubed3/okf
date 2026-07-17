// Package harvester defines metadata harvesting abstractions and files generation utilities for OKF.
package harvester

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
)

// OpenAPIHarvester parses OpenAPI v2 (Swagger) and v3 specifications using getkin/kin-openapi.
type OpenAPIHarvester struct {
	// SpecPath is the filesystem path to the YAML or JSON specification file.
	SpecPath string
}

// NewOpenAPIHarvester creates a new OpenAPIHarvester.
func NewOpenAPIHarvester(specPath string) *OpenAPIHarvester {
	return &OpenAPIHarvester{SpecPath: specPath}
}

// getTypeName extracts a human-readable type string from an OpenAPI schema object, resolving references.
func getTypeName(schemaRef *openapi3.SchemaRef) string {
	if schemaRef == nil {
		return "any"
	}
	if schemaRef.Ref != "" {
		parts := strings.Split(schemaRef.Ref, "/")
		return parts[len(parts)-1]
	}
	s := schemaRef.Value
	if s == nil {
		return "any"
	}
	if s.Type != nil {
		ts := *s.Type
		if len(ts) > 0 {
			if ts[0] == "array" && s.Items != nil {
				return "[]" + getTypeName(s.Items)
			}
			return ts[0]
		}
	}
	return "object"
}

// Harvest parses the OpenAPI file, converting v2 Swagger if needed, and maps endpoints to OKF concepts.
func (h *OpenAPIHarvester) Harvest(ctx context.Context) ([]*bundle.Concept, error) {
	bytes, err := os.ReadFile(h.SpecPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file: %w", err)
	}

	var doc3 *openapi3.T

	// Detect if it is Swagger v2 or OpenAPI v3
	specStr := string(bytes)
	if strings.Contains(specStr, `"swagger":`) || strings.Contains(specStr, "swagger:") {
		// Convert YAML to an intermediate map then re-encode as JSON.
		// openapi2.T's Type field uses openapi3.Types (*[]string) which requires
		// JSON unmarshaling and does not support direct YAML scalar deserialization.
		var raw any
		if err := yaml.Unmarshal(bytes, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse Swagger v2 spec: %w", err)
		}
		jsonBytes, err := json.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to convert Swagger v2 spec to JSON: %w", err)
		}
		var doc2 openapi2.T
		if err := json.Unmarshal(jsonBytes, &doc2); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Swagger v2 spec: %w", err)
		}
		doc3, err = openapi2conv.ToV3(&doc2)
		if err != nil {
			return nil, fmt.Errorf("failed to convert Swagger v2 to OpenAPI v3: %w", err)
		}
	} else {
		// Load OpenAPI v3 natively to resolve relative references
		loader := openapi3.NewLoader()
		doc3, err = loader.LoadFromFile(h.SpecPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse OpenAPI v3 spec: %w", err)
		}
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)
	var concepts []*bundle.Concept

	// Sort paths to ensure deterministic output
	var paths []string
	for p := range doc3.Paths.Map() {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	supportedMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}

	for _, path := range paths {
		pathItem := doc3.Paths.Find(path)
		if pathItem == nil {
			continue
		}

		for _, method := range supportedMethods {
			var op *openapi3.Operation
			switch method {
			case "GET":
				op = pathItem.Get
			case "POST":
				op = pathItem.Post
			case "PUT":
				op = pathItem.Put
			case "DELETE":
				op = pathItem.Delete
			case "PATCH":
				op = pathItem.Patch
			case "OPTIONS":
				op = pathItem.Options
			case "HEAD":
				op = pathItem.Head
			}

			if op == nil {
				continue
			}

			concept := h.buildEndpointConcept(path, method, op, timestamp)
			concepts = append(concepts, concept)
		}
	}

	return concepts, nil
}

// buildEndpointConcept compiles parameter tables and request/response payloads into a formatted markdown endpoint Concept.
func (h *OpenAPIHarvester) buildEndpointConcept(path, method string, op *openapi3.Operation, timestamp string) *bundle.Concept {
	var body strings.Builder

	body.WriteString(fmt.Sprintf("# %s %s\n\n", method, path))

	if op.Summary != "" {
		body.WriteString(fmt.Sprintf("**Summary**: %s\n\n", op.Summary))
	}
	if op.Description != "" {
		body.WriteString(op.Description + "\n\n")
	}

	// 1. Parameters Table
	var params []*openapi3.Parameter
	for _, pRef := range op.Parameters {
		if pRef != nil && pRef.Value != nil {
			params = append(params, pRef.Value)
		}
	}

	if len(params) > 0 {
		body.WriteString("## Parameters\n")
		body.WriteString("| Name | In | Type | Required | Description |\n")
		body.WriteString("| ---- | -- | ---- | -------- | ----------- |\n")
		for _, p := range params {
			typeStr := getTypeName(p.Schema)
			reqStr := "No"
			if p.Required {
				reqStr = "Yes"
			}
			descStr := p.Description
			if descStr == "" {
				descStr = "-"
			}
			// Clean description lines to prevent table breaking
			descStr = strings.ReplaceAll(descStr, "\n", " ")
			body.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n", p.Name, p.In, typeStr, reqStr, descStr))
		}
		body.WriteString("\n")
	}

	// 2. Request Body
	var hasRequestBody bool
	var reqBodyType string
	var reqBodyDesc string

	if op.RequestBody != nil && op.RequestBody.Value != nil {
		reqBody := op.RequestBody.Value
		hasRequestBody = true
		reqBodyDesc = reqBody.Description
		// Try to find application/json or general content
		for mediaType, content := range reqBody.Content {
			if content != nil && content.Schema != nil {
				reqBodyType = fmt.Sprintf("`%s` (%s)", getTypeName(content.Schema), mediaType)
				break
			}
		}
	}

	if hasRequestBody {
		body.WriteString("## Request Body\n")
		if reqBodyType != "" {
			body.WriteString(fmt.Sprintf("- **Type**: %s\n", reqBodyType))
		}
		if reqBodyDesc != "" {
			body.WriteString(fmt.Sprintf("- **Description**: %s\n", strings.ReplaceAll(reqBodyDesc, "\n", " ")))
		}
		body.WriteString("\n")
	}

	// 3. Responses Table
	if len(op.Responses.Map()) > 0 {
		body.WriteString("## Responses\n")
		body.WriteString("| Status Code | Description | Schema Type |\n")
		body.WriteString("| ----------- | ----------- | ----------- |\n")

		var statusCodes []string
		for code := range op.Responses.Map() {
			statusCodes = append(statusCodes, code)
		}
		sort.Strings(statusCodes)

		for _, statusCode := range statusCodes {
			rRef := op.Responses.Map()[statusCode]
			if rRef != nil && rRef.Value != nil {
				r := rRef.Value
				schemaType := "-"
				if r != nil && len(r.Content) > 0 {
					for _, content := range r.Content {
						if content != nil && content.Schema != nil {
							schemaType = getTypeName(content.Schema)
							break
						}
					}
				}
				descStr := ""
				if r != nil && r.Description != nil {
					descStr = *r.Description
				}
				if descStr == "" {
					descStr = "-"
				}
				descStr = strings.ReplaceAll(descStr, "\n", " ")
				body.WriteString(fmt.Sprintf("| %s | %s | %s |\n", statusCode, descStr, schemaType))
			}
		}
		body.WriteString("\n")
	}

	// Make normalized concept ID
	normalizedPath := strings.ReplaceAll(path, "/", "-")
	normalizedPath = strings.ReplaceAll(normalizedPath, "{", "")
	normalizedPath = strings.ReplaceAll(normalizedPath, "}", "")
	normalizedPath = strings.Trim(normalizedPath, "-")

	conceptName := fmt.Sprintf("%s-%s", strings.ToLower(method), normalizedPath)
	conceptID := fmt.Sprintf("endpoints/%s", conceptName)
	conceptPath := fmt.Sprintf("endpoints/%s.md", conceptName)

	title := fmt.Sprintf("%s %s", method, path)
	desc := op.Summary
	if desc == "" {
		desc = op.Description
	}
	if len(desc) > 100 {
		runes := []rune(desc)
		if len(runes) > 100 {
			desc = string(runes[:97]) + "..."
		}
	}
	desc = strings.ReplaceAll(desc, "\n", " ")

	return &bundle.Concept{
		ID:   conceptID,
		Path: conceptPath,
		Frontmatter: bundle.Frontmatter{
			Type:      "API Endpoint",
			Title:     title,
			Desc:      desc,
			Resource:  path,
			Tags:      []string{"api", "endpoint", strings.ToLower(method)},
			Timestamp: timestamp,
		},
		Body: body.String(),
	}
}
