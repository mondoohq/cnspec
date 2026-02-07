// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package lsp

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
	"go.mondoo.com/cnquery/v12"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/mqlc"
	"go.mondoo.com/cnquery/v12/providers"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/resources"
	"k8s.io/utils/ptr"
)

const (
	lsName    = "mql-language-server"
	lsVersion = "0.1.0"
)

func RunStdio() error {
	handler := NewMQLHandler()

	// Create the server using GLSP
	server := server.NewServer(handler, lsName, true)
	return server.RunStdio()
}

// MQLHandler implements the LSP protocol handlers for MQL
type MQLHandler struct {
	protocol.Handler
	mutex sync.RWMutex

	combinedSchema  resources.ResourcesSchema // Combined schema from all providers
	cnqueryFeatures cnquery.Features
	documents       map[protocol.DocumentUri]string      // Cache for document contents
	yamlBundles     map[protocol.DocumentUri]*YAMLBundle // Cache for parsed YAML bundles
}

// NewMQLHandler creates a new MQL language server handler
func NewMQLHandler() *MQLHandler {
	handler := &MQLHandler{
		documents:   make(map[protocol.DocumentUri]string),
		yamlBundles: make(map[protocol.DocumentUri]*YAMLBundle),
	}

	// Use the same approach as cnquery shell - get the combined schema from DefaultRuntime
	runtime := providers.DefaultRuntime()
	handler.combinedSchema = runtime.Schema()

	log.Info().Msg("loaded combined resource schema for MQL language server")

	features, err := cnquery.InitFeatures()
	if err != nil {
		log.Warn().Err(err).Msg("failed to init cnquery features")
		features = cnquery.DefaultFeatures
	}
	handler.cnqueryFeatures = features

	// Debug: log available providers in schema
	handler.logAvailableProviders()

	// Set up the protocol handlers
	handler.Handler = protocol.Handler{
		Initialize:             handler.initialize,
		Initialized:            handler.initialized,
		TextDocumentCompletion: handler.completion,
		CompletionItemResolve:  handler.resolveCompletion,
		TextDocumentDidOpen:    handler.didOpen,
		TextDocumentDidChange:  handler.didChange,
		TextDocumentDidClose:   handler.didClose,
		TextDocumentDidSave:    handler.didSave,
		TextDocumentCodeAction: handler.codeAction,
		TextDocumentHover:      handler.hover,
	}

	return handler
}

func (h *MQLHandler) initialize(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	log.Info().Msg("initializing MQL language server")

	capabilities := protocol.ServerCapabilities{
		TextDocumentSync: &protocol.TextDocumentSyncOptions{
			OpenClose: ptr.To(true),
			Change:    ptr.To(protocol.TextDocumentSyncKindFull),        // Use full sync - simpler and more reliable
			Save:      &protocol.SaveOptions{IncludeText: ptr.To(true)}, // Include text in save events too
		},
		CompletionProvider: &protocol.CompletionOptions{
			TriggerCharacters: []string{".", " "},
			ResolveProvider:   ptr.To(true),
		},
		// Add hover support for displaying type information and documentation
		HoverProvider: ptr.To(true),
		// Add diagnostics support for validation
		// Diagnostics are automatically published when documents change
		// Add code action support for quick fixes
		CodeActionProvider: &protocol.CodeActionOptions{
			CodeActionKinds: []protocol.CodeActionKind{
				protocol.CodeActionKindQuickFix,
				protocol.CodeActionKindSource,
			},
			ResolveProvider: ptr.To(false),
		},
	}

	log.Info().Interface("capabilities", capabilities).Msg("server capabilities configured")

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    lsName,
			Version: ptr.To(lsVersion),
		},
	}, nil
}

func (h *MQLHandler) initialized(context *glsp.Context, params *protocol.InitializedParams) error {
	return nil
}

func (h *MQLHandler) completion(context *glsp.Context, params *protocol.CompletionParams) (any, error) {
	// Check if we're in a YAML bundle
	h.mutex.RLock()
	yamlBundle, isYAML := h.yamlBundles[params.TextDocument.URI]
	h.mutex.RUnlock()

	if isYAML && yamlBundle != nil {
		// Find MQL node at cursor position
		mqlNode := h.findMQLNodeAtPosition(yamlBundle, params.Position)
		if mqlNode == nil {
			// Not in an MQL block, return empty completions
			log.Debug().
				Str("uri", string(params.TextDocument.URI)).
				Uint32("line", params.Position.Line).
				Uint32("char", params.Position.Character).
				Msg("completion: not in MQL block")
			return []protocol.CompletionItem{}, nil
		}

		// Map YAML position to virtual position
		virtualPos := h.mapYAMLToVirtualWithSource(params.Position, mqlNode, yamlBundle.SourceLines)

		// Extract partial MQL for completion
		partialMQL := h.extractPartialMQLForCompletion(mqlNode, virtualPos)

		log.Debug().
			Str("uri", string(params.TextDocument.URI)).
			Str("partialMQL", partialMQL).
			Uint32("yamlLine", params.Position.Line).
			Uint32("yamlChar", params.Position.Character).
			Uint32("virtualLine", virtualPos.Line).
			Uint32("virtualChar", virtualPos.Character).
			Bool("isProp", mqlNode.Context != nil && mqlNode.Context.IsProp).
			Str("mqlContent", mqlNode.Content).
			Msg("🔍 COMPLETION REQUEST: YAML extracted partial MQL")

		if strings.TrimSpace(partialMQL) == "" {
			return h.getCompletions(""), nil
		}

		// Compile with props if available
		var props mqlc.PropsHandler
		if mqlNode.Context != nil && !mqlNode.Context.IsProp && len(mqlNode.Context.QueryProps) > 0 {
			props = &YAMLPropsHandler{
				Props:    mqlNode.Context.QueryProps,
				Schema:   h.combinedSchema,
				Features: h.cnqueryFeatures,
			}
		}

		codeBundle, _ := mqlc.Compile(partialMQL, props, mqlc.NewConfig(h.combinedSchema, h.cnqueryFeatures))
		if codeBundle == nil || len(codeBundle.Suggestions) == 0 {
			log.Debug().
				Str("partialMQL", partialMQL).
				Bool("hasProp", mqlNode.Context != nil && !mqlNode.Context.IsProp && len(mqlNode.Context.QueryProps) > 0).
				Msg("⚠️  COMPLETION: No compiler suggestions, using fallback")
			fallbackItems := h.getCompletions(partialMQL)

			// Add TextEdit to fallback items for YAML context
			lastDotIndex := strings.LastIndex(partialMQL, ".")
			var replaceStartOffset uint32
			if lastDotIndex >= 0 {
				replaceStartOffset = uint32(len(partialMQL) - lastDotIndex - 1)
			} else {
				replaceStartOffset = uint32(len(partialMQL))
			}

			for i := range fallbackItems {
				fallbackItems[i].TextEdit = &protocol.TextEdit{
					Range: protocol.Range{
						Start: protocol.Position{Line: params.Position.Line, Character: params.Position.Character - replaceStartOffset},
						End:   protocol.Position{Line: params.Position.Line, Character: params.Position.Character},
					},
					NewText: fallbackItems[i].Label,
				}
			}

			log.Debug().
				Int("fallbackCount", len(fallbackItems)).
				Msg("✅ COMPLETION: Returning fallback items")
			return fallbackItems, nil
		}

		log.Debug().
			Str("partialMQL", partialMQL).
			Int("count", len(codeBundle.Suggestions)).
			Uint32("yamlLine", params.Position.Line).
			Uint32("yamlChar", params.Position.Character).
			Int("partialMQLLen", len(partialMQL)).
			Uint32("rangeStartChar", params.Position.Character-uint32(len(partialMQL))).
			Msg("✅ COMPLETION: Got compiler suggestions")

		// Log first few suggestions
		for i, suggestion := range codeBundle.Suggestions {
			if i < 5 {
				log.Debug().
					Int("index", i).
					Str("field", suggestion.Field).
					Str("title", suggestion.Title).
					Msg("  → Suggestion")
			}
		}

		// Calculate the range to replace - only replace the last token after the last dot
		// For "asset.name.dow", we want to replace "dow", not the entire "asset.name.dow"
		lastDotIndex := strings.LastIndex(partialMQL, ".")
		var replaceStartOffset uint32
		if lastDotIndex >= 0 {
			// Replace from after the last dot to cursor
			replaceStartOffset = uint32(len(partialMQL) - lastDotIndex - 1)
		} else {
			// No dot found, replace the entire partialMQL
			replaceStartOffset = uint32(len(partialMQL))
		}

		var allItems []protocol.CompletionItem
		for _, suggestion := range codeBundle.Suggestions {
			kind := protocol.CompletionItemKindField
			if strings.Contains(suggestion.Field, "()") || strings.HasSuffix(suggestion.Field, ")") {
				kind = protocol.CompletionItemKindMethod
			}

			item := protocol.CompletionItem{
				Label:         suggestion.Field,
				Kind:          ptr.To(kind),
				Detail:        ptr.To(suggestion.Title),
				Documentation: suggestion.Desc,
				InsertText:    ptr.To(suggestion.Field),
				FilterText:    ptr.To(suggestion.Field), // Help VS Code filter correctly
				// Add TextEdit for YAML context - use YAML document coordinates, not virtual
				// Only replace the last token after the last dot
				TextEdit: &protocol.TextEdit{
					Range: protocol.Range{
						Start: protocol.Position{Line: params.Position.Line, Character: params.Position.Character - replaceStartOffset},
						End:   protocol.Position{Line: params.Position.Line, Character: params.Position.Character},
					},
					NewText: suggestion.Field,
				},
			}
			allItems = append(allItems, item)
		}

		log.Debug().
			Int("itemCount", len(allItems)).
			Msg("📤 COMPLETION: Returning YAML items to VS Code")
		return allItems, nil
	}

	// Original non-YAML completion logic
	query, err := h.getQueryAtPosition(params.TextDocument.URI, params.Position)
	if err != nil {
		log.Debug().
			Str("uri", string(params.TextDocument.URI)).
			Err(err).
			Msg("completion: failed to get query at position")
		return []protocol.CompletionItem{}, nil
	}

	log.Debug().
		Str("uri", string(params.TextDocument.URI)).
		Str("query", query).
		Uint32("line", params.Position.Line).
		Uint32("char", params.Position.Character).
		Msg("🔍 COMPLETION REQUEST: Standalone extracted query")

	// If the query is empty or just whitespace, return basic completions
	if strings.TrimSpace(query) == "" {
		return h.getCompletions(""), nil
	}

	// Compile and get suggestions
	codeBundle, _ := mqlc.Compile(query, nil, mqlc.NewConfig(h.combinedSchema, h.cnqueryFeatures))

	// For completions, we want to ignore compilation errors since the user is still typing
	if codeBundle == nil || len(codeBundle.Suggestions) == 0 {
		log.Debug().
			Str("query", query).
			Msg("⚠️  COMPLETION: No compiler suggestions, using fallback")
		fallbackItems := h.getCompletions(query)
		log.Debug().
			Int("fallbackCount", len(fallbackItems)).
			Msg("✅ COMPLETION: Returning fallback items")
		return fallbackItems, nil
	}

	log.Debug().
		Str("query", query).
		Int("count", len(codeBundle.Suggestions)).
		Msg("✅ COMPLETION: Got compiler suggestions")

	// Log first few suggestions
	for i, suggestion := range codeBundle.Suggestions {
		if i < 5 {
			log.Debug().
				Int("index", i).
				Str("field", suggestion.Field).
				Str("title", suggestion.Title).
				Msg("  → Suggestion")
		}
	}

	var allItems []protocol.CompletionItem
	for _, suggestion := range codeBundle.Suggestions {
		kind := protocol.CompletionItemKindField
		if strings.Contains(suggestion.Field, "()") || strings.HasSuffix(suggestion.Field, ")") {
			kind = protocol.CompletionItemKindMethod
		}

		item := protocol.CompletionItem{
			Label:         suggestion.Field,
			Kind:          ptr.To(kind),
			Detail:        ptr.To(suggestion.Title),
			Documentation: suggestion.Desc,
			InsertText:    ptr.To(suggestion.Field),
			FilterText:    ptr.To(suggestion.Field), // Help VS Code filter correctly
		}
		allItems = append(allItems, item)
	}

	log.Debug().
		Int("itemCount", len(allItems)).
		Msg("📤 COMPLETION: Returning standalone items to VS Code")
	return allItems, nil
}

// getCompletions provides completions for empty or partial queries
func (h *MQLHandler) getCompletions(query string) []protocol.CompletionItem {
	query = strings.TrimSpace(query)

	// If empty query, provide basic resources
	if query == "" {
		basicResources := []string{"asset", "users", "packages", "services", "files", "processes", "ports", "mount", "kernel", "platform"}
		var items []protocol.CompletionItem
		for _, resource := range basicResources {
			items = append(items, protocol.CompletionItem{
				Label:      resource,
				Kind:       ptr.To(protocol.CompletionItemKindClass),
				Detail:     ptr.To("MQL Resource"),
				InsertText: ptr.To(resource),
			})
		}
		return items
	}

	// For partial queries like "users.wh", try to get suggestions from base resource
	parts := strings.Split(query, ".")
	if len(parts) > 1 {
		baseResource := parts[0]
		bundle, err := mqlc.Compile(baseResource, nil, mqlc.NewConfig(h.combinedSchema, h.cnqueryFeatures))
		if err == nil && bundle != nil && len(bundle.Suggestions) > 0 {
			var items []protocol.CompletionItem
			lastPart := parts[len(parts)-1]

			for _, suggestion := range bundle.Suggestions {
				if lastPart == "" || strings.HasPrefix(suggestion.Field, lastPart) {
					kind := protocol.CompletionItemKindField
					if strings.Contains(suggestion.Field, "()") || strings.HasSuffix(suggestion.Field, ")") {
						kind = protocol.CompletionItemKindMethod
					}
					items = append(items, protocol.CompletionItem{
						Label:         suggestion.Field,
						Kind:          ptr.To(kind),
						Detail:        ptr.To(suggestion.Title),
						Documentation: suggestion.Desc,
						InsertText:    ptr.To(suggestion.Field),
					})
				}
			}
			if len(items) > 0 {
				return items
			}
		}
	}

	// Fallback to basic completions
	return h.getCompletions("")
}

// getQueryAtPosition extracts the MQL query from the document up to the given position
func (h *MQLHandler) getQueryAtPosition(uri protocol.DocumentUri, position protocol.Position) (string, error) {
	h.mutex.RLock()
	content, exists := h.documents[uri]
	h.mutex.RUnlock()

	if !exists {
		return "", errors.New("document not found in cache")
	}

	// Extract MQL query based on file type
	return h.extractMQLQuery(content, position)
}

// extractMQLQuery extracts the MQL query from content up to the given position
func (h *MQLHandler) extractMQLQuery(content string, position protocol.Position) (string, error) {
	// For YAML files, we should be using the new YAML bundle approach in the caller
	// This method is kept for plain MQL files

	// Original logic for non-YAML files
	lines := strings.Split(content, "\n")

	if int(position.Line) >= len(lines) {
		return "", errors.New("position line exceeds document length")
	}

	// Get the text up to the cursor position
	var queryLines []string

	// Add all complete lines before the cursor line
	for i := 0; i < int(position.Line); i++ {
		queryLines = append(queryLines, lines[i])
	}

	// Add the partial line up to the cursor position
	currentLine := lines[position.Line]
	if int(position.Character) > len(currentLine) {
		queryLines = append(queryLines, currentLine)
	} else {
		queryLines = append(queryLines, currentLine[:position.Character])
	}

	fullQuery := strings.Join(queryLines, "\n")

	log.Debug().Str("full_query_before_extraction", fullQuery).Int("position_line", int(position.Line)).Int("position_char", int(position.Character)).Msg("query extraction details")

	// Determine file type and extract MQL accordingly
	return h.extractMQLFromContent(fullQuery, "extraction"), nil
}

// extractMQLFromContent extracts MQL from content based on file type
// This is kept for backward compatibility with plain MQL files
func (h *MQLHandler) extractMQLFromContent(content string, context string) string {
	// Default: treat as pure MQL
	return content
}

// isYAMLPolicy checks if the content looks like a YAML policy file
func (h *MQLHandler) isYAMLPolicy(content string) bool {
	// Quick heuristic: check for queries array and mql fields
	return strings.Contains(content, "queries:") && strings.Contains(content, "mql:")
}

// extractExpressionAtPosition extracts the MQL expression at the given position
func (h *MQLHandler) extractExpressionAtPosition(line string, charPos int) string {
	if charPos > len(line) {
		charPos = len(line)
	}

	start := charPos
	end := charPos

	// Special case: if cursor is on a block character, find the expression before it
	if start < len(line) && (line[start] == '{' || line[start] == '}') {
		// Find the expression before the block
		blockStart := start
		for blockStart > 0 && (line[blockStart-1] == ' ' || line[blockStart-1] == '\t') {
			blockStart--
		}
		end = blockStart
		start = blockStart
		for start > 0 && (isIdentifierChar(line[start-1]) || line[start-1] == '.') {
			start--
		}
		if start < end {
			return line[start:end] + ".{}"
		}
	}

	// Expand backwards to find the start of the expression
	for start > 0 {
		ch := line[start-1]
		if isIdentifierChar(ch) || ch == '.' || ch == '(' || ch == ')' {
			start--
		} else {
			break
		}
	}

	// Expand forwards to find the end of the expression
	for end < len(line) {
		ch := line[end]
		if isIdentifierChar(ch) || ch == '.' {
			end++
		} else if ch == '(' {
			// Include method calls - find the closing parenthesis
			parenCount := 1
			end++
			for end < len(line) && parenCount > 0 {
				switch line[end] {
				case '(':
					parenCount++
				case ')':
					parenCount--
				}
				end++
			}
		} else {
			break
		}
	}

	if start >= end {
		return ""
	}
	return line[start:end]
}

// getQueryWithCursorExpression builds a query that includes the full expression at the cursor position
func (h *MQLHandler) getQueryWithCursorExpression(uri protocol.DocumentUri, position protocol.Position) (string, error) {
	h.mutex.RLock()
	content, exists := h.documents[uri]
	h.mutex.RUnlock()

	if !exists {
		return "", errors.New("document not found in cache")
	}

	// For YAML files, the hover function now handles them directly, so this shouldn't be called
	// This is kept for plain MQL files only

	// Original logic for non-YAML files
	lines := strings.Split(content, "\n")
	if int(position.Line) >= len(lines) {
		return "", errors.New("position line exceeds document length")
	}

	// Extract the expression at the cursor position
	currentLine := lines[position.Line]
	expression := h.extractExpressionAtPosition(currentLine, int(position.Character))
	if expression == "" {
		return "", errors.New("no expression found at cursor")
	}

	return expression, nil
}

// Document lifecycle methods
func (h *MQLHandler) didOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	h.mutex.Lock()
	h.documents[params.TextDocument.URI] = params.TextDocument.Text

	// Parse YAML bundle if applicable
	if h.isYAMLPolicy(params.TextDocument.Text) {
		bundle, err := h.parseYAMLBundle(params.TextDocument.Text, params.TextDocument.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", string(params.TextDocument.URI)).Msg("failed to parse YAML bundle")
		} else {
			h.yamlBundles[params.TextDocument.URI] = bundle
			h.logYAMLBundleInfo(bundle)
		}
	}
	h.mutex.Unlock()

	// Validate the document and publish diagnostics
	go h.validateDocument(context, params.TextDocument.URI)
	return nil
}

func (h *MQLHandler) didChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	if len(params.ContentChanges) == 0 {
		return nil
	}

	h.mutex.Lock()

	// With full document sync, we expect the entire document content in the first change
	var updatedContent string
	for _, change := range params.ContentChanges {
		switch v := change.(type) {
		case protocol.TextDocumentContentChangeEvent:
			h.documents[params.TextDocument.URI] = v.Text
			updatedContent = v.Text
		case protocol.TextDocumentContentChangeEventWhole:
			h.documents[params.TextDocument.URI] = v.Text
			updatedContent = v.Text
		case map[string]interface{}:
			if text, ok := v["text"].(string); ok {
				h.documents[params.TextDocument.URI] = text
				updatedContent = text
				break
			}
		}
	}

	// Re-parse YAML bundle if applicable
	if updatedContent != "" && h.isYAMLPolicy(updatedContent) {
		bundle, err := h.parseYAMLBundle(updatedContent, params.TextDocument.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", string(params.TextDocument.URI)).Msg("failed to parse YAML bundle")
			// Remove invalid bundle from cache
			delete(h.yamlBundles, params.TextDocument.URI)
		} else {
			h.yamlBundles[params.TextDocument.URI] = bundle
		}
	} else {
		// Not a YAML bundle, remove from cache if it was there
		delete(h.yamlBundles, params.TextDocument.URI)
	}

	h.mutex.Unlock()

	// Validate the document and publish diagnostics
	go func() {
		time.Sleep(50 * time.Millisecond) // Small delay for document processing
		h.validateDocument(context, params.TextDocument.URI)
	}()

	return nil
}

func (h *MQLHandler) didClose(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	delete(h.documents, params.TextDocument.URI)
	delete(h.yamlBundles, params.TextDocument.URI)
	return nil
}

func (h *MQLHandler) resolveCompletion(context *glsp.Context, params *protocol.CompletionItem) (*protocol.CompletionItem, error) {
	return params, nil
}

// didSave handles document save events and triggers validation
func (h *MQLHandler) didSave(context *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
	log.Debug().Str("uri", string(params.TextDocument.URI)).Msg("document saved")

	// If the save event includes text, update our cache
	if params.Text != nil {
		h.mutex.Lock()
		h.documents[params.TextDocument.URI] = *params.Text
		h.mutex.Unlock()
	}

	// Validate the document and publish diagnostics
	go h.validateDocument(context, params.TextDocument.URI)

	return nil
}

// validateDocument validates the MQL content and publishes diagnostics
func (h *MQLHandler) validateDocument(context *glsp.Context, uri protocol.DocumentUri) {
	h.mutex.RLock()
	content, exists := h.documents[uri]
	yamlBundle, isYAML := h.yamlBundles[uri]
	h.mutex.RUnlock()

	if !exists {
		return
	}

	diagnostics := make([]protocol.Diagnostic, 0)
	// Handle YAML policy files with new implementation
	if isYAML && yamlBundle != nil {
		h.validateYAMLBundle(yamlBundle, &diagnostics)
	} else {
		// Plain MQL file
		mqlContent := h.extractMQLFromContent(content, "validation")
		h.validateMQL(mqlContent, &diagnostics)
	}

	// Publish diagnostics to the client
	context.Notify("textDocument/publishDiagnostics", &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})
}

// validateYAMLBundle validates all MQL nodes in a YAML bundle
func (h *MQLHandler) validateYAMLBundle(bundle *YAMLBundle, diagnostics *[]protocol.Diagnostic) {
	for _, mqlNode := range bundle.MQLNodes {
		h.validateYAMLMQLNode(mqlNode, diagnostics)
	}
}

// validateYAMLMQLNode validates a single MQL node from YAML
func (h *MQLHandler) validateYAMLMQLNode(mqlNode *MQLNode, diagnostics *[]protocol.Diagnostic) {
	// Compile with props if this is a main query
	var props mqlc.PropsHandler
	if mqlNode.Context != nil && !mqlNode.Context.IsProp && len(mqlNode.Context.QueryProps) > 0 {
		props = &YAMLPropsHandler{
			Props:    mqlNode.Context.QueryProps,
			Schema:   h.combinedSchema,
			Features: h.cnqueryFeatures,
		}
	}

	_, compileErr := mqlc.Compile(mqlNode.Content, props, mqlc.NewConfig(h.combinedSchema, h.cnqueryFeatures))
	if compileErr != nil {
		// Create diagnostic range for the entire MQL node
		startLine := uint32(mqlNode.StartLine)
		endLine := uint32(mqlNode.EndLine)

		// For block scalars, adjust to skip the "mql: |" line
		if mqlNode.IsBlock {
			startLine++
		}

		diagnostic := protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: startLine, Character: uint32(mqlNode.StartCol)},
				End:   protocol.Position{Line: endLine, Character: 0},
			},
			Severity: ptr.To(protocol.DiagnosticSeverityError),
			Source:   ptr.To("mql-compiler"),
			Message:  compileErr.Error(),
		}
		*diagnostics = append(*diagnostics, diagnostic)
	}
}

func (h *MQLHandler) validateMQL(mqlContent string, diagnostics *[]protocol.Diagnostic) {
	// Compile the MQL and check for errors
	_, compileErr := mqlc.Compile(mqlContent, nil, mqlc.NewConfig(h.combinedSchema, h.cnqueryFeatures))

	if compileErr != nil {
		// Calculate end position more accurately
		lines := strings.Split(mqlContent, "\n")
		lastLine := uint32(len(lines) - 1)
		lastLineLength := uint32(0)
		if len(lines) > 0 {
			lastLineLength = uint32(len(lines[len(lines)-1]))
		}

		diagnostic := protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: lastLine, Character: lastLineLength},
			},
			Severity: ptr.To(protocol.DiagnosticSeverityError),
			Source:   ptr.To("mql-compiler"),
			Message:  compileErr.Error(),
		}

		*diagnostics = append(*diagnostics, diagnostic)
	}
}

// codeAction provides quick fixes and code actions for diagnostics
func (h *MQLHandler) codeAction(context *glsp.Context, params *protocol.CodeActionParams) (any, error) {
	var actions []protocol.CodeAction

	// If there are diagnostics in the range, try to provide quick fixes
	for _, diagnostic := range params.Context.Diagnostics {
		if diagnostic.Source != nil && *diagnostic.Source == "mql-compiler" {
			h.mutex.RLock()
			content, exists := h.documents[params.TextDocument.URI]
			h.mutex.RUnlock()

			if !exists {
				continue
			}

			// Try to get suggestions by compiling the content
			mqlContent := h.extractMQLFromContent(content, "codeaction")
			bundle, _ := mqlc.Compile(mqlContent, nil, mqlc.NewConfig(h.combinedSchema, h.cnqueryFeatures))

			if bundle != nil && len(bundle.Suggestions) > 0 {
				// Create quick fix actions for each suggestion
				for _, suggestion := range bundle.Suggestions {
					action := protocol.CodeAction{
						Title: fmt.Sprintf("Replace with '%s'", suggestion.Field),
						Kind:  ptr.To(protocol.CodeActionKindQuickFix),
						Edit: &protocol.WorkspaceEdit{
							Changes: map[protocol.DocumentUri][]protocol.TextEdit{
								params.TextDocument.URI: {
									{
										Range:   diagnostic.Range,
										NewText: suggestion.Field,
									},
								},
							},
						},
						Diagnostics: []protocol.Diagnostic{diagnostic},
					}
					actions = append(actions, action)
				}
			}
		}
	}

	return actions, nil
}

// hover provides type information and documentation for MQL expressions
func (h *MQLHandler) hover(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	// Check if we're in a YAML bundle
	h.mutex.RLock()
	yamlBundle, isYAML := h.yamlBundles[params.TextDocument.URI]
	h.mutex.RUnlock()

	if isYAML && yamlBundle != nil {
		// Find MQL node at cursor position
		mqlNode := h.findMQLNodeAtPosition(yamlBundle, params.Position)
		if mqlNode == nil {
			return nil, nil
		}

		// Map YAML position to virtual position
		virtualPos := h.mapYAMLToVirtualWithSource(params.Position, mqlNode, yamlBundle.SourceLines)

		// Extract the expression at the cursor in the virtual document
		lines := strings.Split(mqlNode.Content, "\n")
		if int(virtualPos.Line) >= len(lines) {
			return nil, nil
		}

		currentLine := lines[virtualPos.Line]
		expression := h.extractExpressionAtPosition(currentLine, int(virtualPos.Character))
		if expression == "" {
			return nil, nil
		}

		// Compile with props if available
		var props mqlc.PropsHandler
		if mqlNode.Context != nil && !mqlNode.Context.IsProp && len(mqlNode.Context.QueryProps) > 0 {
			props = &YAMLPropsHandler{
				Props:    mqlNode.Context.QueryProps,
				Schema:   h.combinedSchema,
				Features: h.cnqueryFeatures,
			}
		}

		codeBundle, _ := mqlc.Compile(expression, props, mqlc.NewConfig(h.combinedSchema, h.cnqueryFeatures))
		if codeBundle == nil {
			return nil, nil
		}

		hoverInfo := h.analyzeCompiledCode(codeBundle, expression)
		if hoverInfo == "" {
			return nil, nil
		}

		// Calculate hover range in virtual document, then map back to YAML
		virtualRange := h.calculateHoverRangeFromLine(currentLine, int(virtualPos.Character))
		if virtualRange == nil {
			return nil, nil
		}

		// Adjust virtual range to be relative to the virtual document
		virtualRange.Start.Line = virtualPos.Line
		virtualRange.End.Line = virtualPos.Line

		yamlRange := h.mapRangeVirtualToYAML(*virtualRange, mqlNode)

		return &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: hoverInfo,
			},
			Range: &yamlRange,
		}, nil
	}

	// Original non-YAML hover logic
	query, err := h.getQueryWithCursorExpression(params.TextDocument.URI, params.Position)
	if err != nil {
		return nil, nil
	}

	// Compile the query to get rich type information
	codeBundle, _ := mqlc.Compile(query, nil, mqlc.NewConfig(h.combinedSchema, h.cnqueryFeatures))
	if codeBundle == nil {
		return nil, nil
	}

	// Analyze the compiled code to extract hover information
	hoverInfo := h.analyzeCompiledCode(codeBundle, query)
	if hoverInfo == "" {
		return nil, nil
	}

	hoverRange := h.calculateHoverRange(params.TextDocument.URI, params.Position)

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: hoverInfo,
		},
		Range: hoverRange,
	}, nil
}

// calculateHoverRangeFromLine calculates hover range within a single line
func (h *MQLHandler) calculateHoverRangeFromLine(line string, charPos int) *protocol.Range {
	if charPos >= len(line) {
		return nil
	}

	// Find word boundaries around the cursor position
	start := charPos
	end := charPos

	// Find start of word
	for start > 0 && isIdentifierChar(line[start-1]) {
		start--
	}

	// Find end of word
	for end < len(line) && isIdentifierChar(line[end]) {
		end++
	}

	return &protocol.Range{
		Start: protocol.Position{Line: 0, Character: uint32(start)},
		End:   protocol.Position{Line: 0, Character: uint32(end)},
	}
}

// analyzeCompiledCode extracts basic information from the compiled MQL bundle
func (h *MQLHandler) analyzeCompiledCode(bundle *llx.CodeBundle, query string) string {
	var info strings.Builder

	if bundle.CodeV2 == nil || len(bundle.CodeV2.Blocks) == 0 {
		return ""
	}

	mainBlock := bundle.CodeV2.Blocks[0]

	// Get entrypoints (final results)
	entrypoints := bundle.CodeV2.Entrypoints()
	if len(entrypoints) > 0 {
		for _, ep := range entrypoints {
			chunk := bundle.CodeV2.Chunk(ep)
			if chunk != nil {
				resultType := chunk.DereferencedTypeV2(bundle.CodeV2)
				typeLabel := resultType.Label()

				// Critical edge case: If this is a phantom block but no explicit {} in query,
				// infer the actual type from the chain (e.g., cisco.iosxr.interfaces -> []cisco.iosxr.interface)
				if typeLabel == "block" && !strings.Contains(query, "{") {
					inferredType := h.inferTypeFromChain(mainBlock, bundle.CodeV2)
					if inferredType != "" {
						typeLabel = inferredType
					}
				}

				info.WriteString(fmt.Sprintf("**Type**: `%s`\n\n", typeLabel))

				// Try to identify the provider
				provider := h.identifyProvider(chunk, bundle.CodeV2)
				if provider != "" {
					info.WriteString(fmt.Sprintf("**Provider**: `%s`\n\n", provider))
				}
			}
		}
	}

	// Show execution chain for debugging complex queries
	if len(mainBlock.Chunks) > 1 {
		chainInfo := h.buildExecutionChain(mainBlock, bundle.CodeV2, query)
		if chainInfo != "" {
			info.WriteString("**Chain**:\n")
			info.WriteString(chainInfo)
			info.WriteString("\n")
		}
	}

	// Add resource documentation if available
	resourceDocs := h.getResourceDocumentation(query)
	if resourceDocs != "" {
		info.WriteString("**Documentation**:\n")
		info.WriteString(resourceDocs)
	}

	return info.String()
}

// inferTypeFromChain infers the actual type from the execution chain for block results
func (h *MQLHandler) inferTypeFromChain(mainBlock *llx.Block, code *llx.CodeV2) string {
	if len(mainBlock.Chunks) < 2 {
		return ""
	}

	// Look at the second-to-last chunk to infer the underlying type
	prevChunk := mainBlock.Chunks[len(mainBlock.Chunks)-2]
	if prevChunk != nil {
		prevType := prevChunk.DereferencedTypeV2(code)
		prevTypeLabel := prevType.Label()

		// If the previous type is an array, try to infer the element type
		if strings.HasPrefix(prevTypeLabel, "[]") {
			// Extract the element type from the array type
			elementType := strings.TrimPrefix(prevTypeLabel, "[]")
			if elementType != "block" && elementType != "" {
				return "[]" + elementType
			}
		}

		// For non-array types, check if we can build a more specific type
		if prevTypeLabel != "block" && prevTypeLabel != "" {
			// Try to build the type from the resource chain
			if resourceType := h.buildResourceTypeFromChain(mainBlock.Chunks, code); resourceType != "" {
				return resourceType
			}
			return prevTypeLabel
		}
	}

	return ""
}

// buildResourceTypeFromChain builds a specific resource type from the execution chain
func (h *MQLHandler) buildResourceTypeFromChain(chunks []*llx.Chunk, code *llx.CodeV2) string {
	var parts []string

	for _, chunk := range chunks {
		if chunk.Id != "" && chunk.Id != "{}" {
			// Check if this is a known resource
			if resourceInfo := h.combinedSchema.Lookup(chunk.Id); resourceInfo != nil {
				parts = append(parts, chunk.Id)
			}
		}
	}

	if len(parts) >= 2 {
		// Build a type like "[]cisco.iosxr.interface" from parts
		return "[]" + strings.Join(parts, ".")
	}

	return ""
}

// buildExecutionChain creates a simplified execution chain for complex queries
func (h *MQLHandler) buildExecutionChain(mainBlock *llx.Block, code *llx.CodeV2, query string) string {
	var chainParts []string

	for i, chunk := range mainBlock.Chunks {
		// Skip phantom {} blocks unless they're explicitly in the query
		if chunk.Id == "{}" && !strings.Contains(query, "{") {
			continue
		}

		chunkDesc := h.getChunkDescription(chunk, i)
		if chunkDesc != "" {
			chainParts = append(chainParts, fmt.Sprintf("%d. %s", len(chainParts)+1, chunkDesc))
		}
	}

	if len(chainParts) > 0 {
		return strings.Join(chainParts, "\n")
	}
	return ""
}

// getChunkDescription provides a concise description of what a chunk does
func (h *MQLHandler) getChunkDescription(chunk *llx.Chunk, index int) string {
	if chunk.Id != "" {
		if chunk.Function != nil && chunk.Function.Binding != 0 {
			return fmt.Sprintf("Call `%s`", chunk.Id)
		} else {
			return fmt.Sprintf("Reference `%s`", chunk.Id)
		}
	}

	if chunk.Call == llx.Chunk_PRIMITIVE && chunk.Primitive != nil {
		return "Literal value"
	}

	return "Operation"
}

// identifyProvider tries to determine which provider a chunk belongs to
func (h *MQLHandler) identifyProvider(chunk *llx.Chunk, code *llx.CodeV2) string {
	// Try to identify from resource name
	if chunk.Id != "" {
		if resourceInfo := h.combinedSchema.Lookup(chunk.Id); resourceInfo != nil {
			return resourceInfo.Provider
		}
	}

	// Try to identify from function bindings
	if chunk.Function != nil && chunk.Function.Binding != 0 {
		bindingChunk := code.Chunk(chunk.Function.Binding)
		if bindingChunk != nil && bindingChunk.Id != "" {
			if resourceInfo := h.combinedSchema.Lookup(bindingChunk.Id); resourceInfo != nil {
				return resourceInfo.Provider
			}
		}
	}

	return ""
}

// getResourceDocumentation extracts documentation for resources from the schema
func (h *MQLHandler) getResourceDocumentation(query string) string {
	// Extract resource names from the query
	words := strings.Fields(strings.ReplaceAll(query, ".", " "))
	var docs strings.Builder
	seen := make(map[string]bool)

	for _, word := range words {
		word = strings.Trim(word, "()[]{}\"'`")
		if word == "" || seen[word] {
			continue
		}

		if resourceInfo := h.combinedSchema.Lookup(word); resourceInfo != nil {
			seen[word] = true
			if resourceInfo.Title != "" || resourceInfo.Desc != "" {
				docs.WriteString(fmt.Sprintf("**%s**", word))
				if resourceInfo.Provider != "" {
					docs.WriteString(fmt.Sprintf(" (from `%s`)", resourceInfo.Provider))
				}
				docs.WriteString("\n")

				if resourceInfo.Title != "" {
					docs.WriteString(fmt.Sprintf("*%s*\n", resourceInfo.Title))
				}
				if resourceInfo.Desc != "" {
					docs.WriteString(fmt.Sprintf("%s\n", resourceInfo.Desc))
				}
				docs.WriteString("\n")
			}
		}
	}

	return docs.String()
}

// calculateHoverRange determines the range of text that the hover applies to
func (h *MQLHandler) calculateHoverRange(uri protocol.DocumentUri, position protocol.Position) *protocol.Range {
	h.mutex.RLock()
	content, exists := h.documents[uri]
	h.mutex.RUnlock()

	if !exists {
		return nil
	}

	lines := strings.Split(content, "\n")
	if int(position.Line) >= len(lines) {
		return nil
	}

	line := lines[position.Line]
	if int(position.Character) >= len(line) {
		return nil
	}

	// Find word boundaries around the cursor position
	start := int(position.Character)
	end := int(position.Character)

	// Find start of word
	for start > 0 && isIdentifierChar(line[start-1]) {
		start--
	}

	// Find end of word
	for end < len(line) && isIdentifierChar(line[end]) {
		end++
	}

	return &protocol.Range{
		Start: protocol.Position{Line: position.Line, Character: uint32(start)},
		End:   protocol.Position{Line: position.Line, Character: uint32(end)},
	}
}

// isIdentifierChar checks if a character is valid in MQL identifiers
func isIdentifierChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}

// logAvailableProviders logs which providers are available in the schema for debugging
func (h *MQLHandler) logAvailableProviders() {
	providerMap := make(map[string]int)

	// Iterate through all resources to count by provider
	allResources := h.combinedSchema.AllResources()
	for _, resourceInfo := range allResources {
		if resourceInfo.Provider != "" {
			providerMap[resourceInfo.Provider]++
		} else {
			providerMap["unknown"]++
		}
	}

	log.Info().
		Interface("providers", providerMap).
		Int("total_resources", len(allResources)).
		Msg("available providers in combined schema")
}
