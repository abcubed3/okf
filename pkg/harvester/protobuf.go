// Package harvester defines metadata harvesting abstractions and files generation utilities for OKF.
package harvester

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/abcubed3/okf/pkg/bundle"
	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ProtobufHarvester extracts metadata from .proto files using the bufbuild/protocompile library.
type ProtobufHarvester struct {
	// Path is the directory or file path to walk/parse.
	Path string
}

// NewProtobufHarvester creates a new ProtobufHarvester.
func NewProtobufHarvester(path string) *ProtobufHarvester {
	return &ProtobufHarvester{Path: path}
}

// Harvest scans the path and generates OKF Concepts.
func (h *ProtobufHarvester) Harvest(ctx context.Context) ([]*bundle.Concept, error) {
	absPath, err := filepath.Abs(h.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("path does not exist: %w", err)
	}

	var importPath string
	var filesToCompile []string

	if info.IsDir() {
		importPath = absPath
		err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Ext(path) == ".proto" {
				rel, err := filepath.Rel(importPath, path)
				if err == nil {
					filesToCompile = append(filesToCompile, rel)
				}
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to walk directory: %w", err)
		}
	} else {
		importPath = filepath.Dir(absPath)
		filesToCompile = append(filesToCompile, filepath.Base(absPath))
	}

	if len(filesToCompile) == 0 {
		return nil, nil
	}

	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{
			ImportPaths: []string{importPath},
		},
		SourceInfoMode: protocompile.SourceInfoExtraComments,
	}

	fds, err := compiler.Compile(ctx, filesToCompile...)
	if err != nil {
		return nil, fmt.Errorf("protobuf compilation failed: %w", err)
	}

	var concepts []*bundle.Concept
	timestamp := time.Now().UTC().Format(time.RFC3339)

	for _, fd := range fds {
		concepts = append(concepts, h.extractConcepts(fd, timestamp)...)
	}

	return concepts, nil
}

// extractConcepts converts a compiled FileDescriptor into message and service Concepts.
func (h *ProtobufHarvester) extractConcepts(fd linker.File, timestamp string) []*bundle.Concept {
	var concepts []*bundle.Concept
	pkg := string(fd.Package())

	// 1. Process Messages (collecting all nested messages as well)
	allMsgs := collectAllMessages(fd.Messages())
	for _, msg := range allMsgs {
		concepts = append(concepts, h.buildMessageConcept(pkg, fd, msg, timestamp))
	}

	// 2. Process Services
	srvs := fd.Services()
	for i := 0; i < srvs.Len(); i++ {
		srv := srvs.Get(i)
		concepts = append(concepts, h.buildServiceConcept(pkg, fd, srv, timestamp))
	}

	return concepts
}

// collectAllMessages recursively traverses messages to extract nested message declarations.
func collectAllMessages(msgs protoreflect.MessageDescriptors) []protoreflect.MessageDescriptor {
	var results []protoreflect.MessageDescriptor
	for i := 0; i < msgs.Len(); i++ {
		m := msgs.Get(i)
		results = append(results, m)
		results = append(results, collectAllMessages(m.Messages())...)
	}
	return results
}

// buildMessageConcept maps a parsed protoreflect.MessageDescriptor into a formatted markdown message Concept.
func (h *ProtobufHarvester) buildMessageConcept(pkg string, fd linker.File, msg protoreflect.MessageDescriptor, timestamp string) *bundle.Concept {
	var body strings.Builder
	msgName := string(msg.Name())
	fqn := string(msg.FullName())

	body.WriteString(fmt.Sprintf("# Protobuf Message %s\n\n", fqn))

	desc := getComments(fd, msg)
	if desc != "" {
		body.WriteString(desc)
		body.WriteString("\n\n")
	}

	body.WriteString("## Fields\n")
	body.WriteString("| Name | Type | Tag | Description |\n")
	body.WriteString("| ---- | ---- | --- | ----------- |\n")

	fields := msg.Fields()
	for i := 0; i < fields.Len(); i++ {
		f := fields.Get(i)
		fName := string(f.Name())

		var typeStr string
		if f.IsMap() {
			keyType := string(f.MapKey().Kind().String())
			valType := string(f.MapValue().Kind().String())
			if f.MapValue().Kind() == protoreflect.MessageKind {
				valType = string(f.MapValue().Message().FullName())
			}
			typeStr = fmt.Sprintf("map<%s, %s>", keyType, valType)
		} else {
			if f.Kind() == protoreflect.MessageKind {
				typeStr = string(f.Message().FullName())
			} else {
				typeStr = f.Kind().String()
			}
			if f.IsList() {
				typeStr = "repeated " + typeStr
			}
		}

		fTag := fmt.Sprintf("%d", f.Number())
		fDesc := getComments(fd, f)
		if fDesc == "" {
			fDesc = "-"
		}
		fDesc = strings.ReplaceAll(fDesc, "\n", " ")

		body.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", fName, typeStr, fTag, fDesc))
	}
	body.WriteString("\n")

	conceptName := strings.ToLower(fqn)
	conceptID := fmt.Sprintf("protobuf/%s", conceptName)
	conceptPath := fmt.Sprintf("protobuf/%s.md", conceptName)

	shortDesc := desc
	if shortDesc == "" {
		shortDesc = fmt.Sprintf("Protobuf message definition for %s.", msgName)
	}
	if len(shortDesc) > 100 {
		runes := []rune(shortDesc)
		if len(runes) > 100 {
			shortDesc = string(runes[:97]) + "..."
		}
	}
	shortDesc = strings.ReplaceAll(shortDesc, "\n", " ")

	return &bundle.Concept{
		ID:   conceptID,
		Path: conceptPath,
		Frontmatter: bundle.Frontmatter{
			Type:      "Protobuf Message",
			Title:     fmt.Sprintf("Message %s", msgName),
			Desc:      shortDesc,
			Resource:  fqn,
			Tags:      []string{"protobuf", "message", pkg},
			Timestamp: timestamp,
		},
		Body: body.String(),
	}
}

// buildServiceConcept maps a parsed protoreflect.ServiceDescriptor into a formatted markdown service Concept.
func (h *ProtobufHarvester) buildServiceConcept(pkg string, fd linker.File, srv protoreflect.ServiceDescriptor, timestamp string) *bundle.Concept {
	var body strings.Builder
	srvName := string(srv.Name())
	fqn := string(srv.FullName())

	body.WriteString(fmt.Sprintf("# Protobuf Service %s\n\n", fqn))

	desc := getComments(fd, srv)
	if desc != "" {
		body.WriteString(desc)
		body.WriteString("\n\n")
	}

	body.WriteString("## RPC Methods\n")
	body.WriteString("| Method | Request | Response | Description |\n")
	body.WriteString("| ------ | ------- | -------- | ----------- |\n")

	methods := srv.Methods()
	for i := 0; i < methods.Len(); i++ {
		r := methods.Get(i)
		rName := string(r.Name())
		reqFQN := string(r.Input().FullName())
		respFQN := string(r.Output().FullName())

		rDesc := getComments(fd, r)
		if rDesc == "" {
			rDesc = "-"
		}
		rDesc = strings.ReplaceAll(rDesc, "\n", " ")

		body.WriteString(fmt.Sprintf("| %s | [%s](%s) | [%s](%s) | %s |\n",
			rName,
			reqFQN, resolveProtoLink(reqFQN),
			respFQN, resolveProtoLink(respFQN),
			rDesc))
	}
	body.WriteString("\n")

	conceptName := strings.ToLower(fqn)
	conceptID := fmt.Sprintf("protobuf/%s", conceptName)
	conceptPath := fmt.Sprintf("protobuf/%s.md", conceptName)

	shortDesc := desc
	if shortDesc == "" {
		shortDesc = fmt.Sprintf("Protobuf service definition for %s.", srvName)
	}
	if len(shortDesc) > 100 {
		runes := []rune(shortDesc)
		if len(runes) > 100 {
			shortDesc = string(runes[:97]) + "..."
		}
	}
	shortDesc = strings.ReplaceAll(shortDesc, "\n", " ")

	return &bundle.Concept{
		ID:   conceptID,
		Path: conceptPath,
		Frontmatter: bundle.Frontmatter{
			Type:      "Protobuf Service",
			Title:     fmt.Sprintf("Service %s", srvName),
			Desc:      shortDesc,
			Resource:  fqn,
			Tags:      []string{"protobuf", "service", pkg},
			Timestamp: timestamp,
		},
		Body: body.String(),
	}
}

// resolveProtoLink generates target concept file names for links between protobuf components.
func resolveProtoLink(fqn string) string {
	return strings.ToLower(fqn) + ".md"
}

// getComments retrieves clean documentation comments from the protobuf SourceLocations.
func getComments(file linker.File, desc protoreflect.Descriptor) string {
	loc := file.SourceLocations().ByDescriptor(desc)
	comment := strings.TrimSpace(loc.LeadingComments)
	if comment == "" {
		comment = strings.TrimSpace(loc.TrailingComments)
	}
	return comment
}
