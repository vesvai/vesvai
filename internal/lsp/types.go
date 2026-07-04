package lsp

import (
	"encoding/json"
	"strconv"
)

const (
	jsonrpcVersion = "2.0"
)

type RequestID struct {
	intVal   int64
	strVal   string
	isString bool
}

func IntID(id int64) RequestID {
	return RequestID{intVal: id, isString: false}
}

func StringID(id string) RequestID {
	return RequestID{strVal: id, isString: true}
}

func (r RequestID) MarshalJSON() ([]byte, error) {
	if r.isString {
		return json.Marshal(r.strVal)
	}
	return json.Marshal(r.intVal)
}

func (r *RequestID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		r.strVal = s
		r.isString = true
		return nil
	}
	var n int64
	if err := json.Unmarshal(data, &n); err == nil {
		r.intVal = n
		r.isString = false
		return nil
	}
	return nil
}

func (r RequestID) String() string {
	if r.isString {
		return r.strVal
	}
	return strconv.FormatInt(r.intVal, 10)
}

type Request struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      RequestID `json:"id"`
	Method  string    `json:"method"`
	Params  any       `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      RequestID `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
}

type Notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return e.Message
}

const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternalError  = -32603
)

const (
	ProtocolVersion = "3.17.0"
	ClientName      = "vesvai"
	ClientVersion   = "0.1.0"
)

const (
	MethodInitialize                     = "initialize"
	MethodInitialized                    = "initialized"
	MethodShutdown                       = "shutdown"
	MethodExit                           = "exit"
	MethodTextDocumentDidOpen            = "textDocument/didOpen"
	MethodTextDocumentDidClose           = "textDocument/didClose"
	MethodTextDocumentDidChange          = "textDocument/didChange"
	MethodTextDocumentDidSave            = "textDocument/didSave"
	MethodTextDocumentCompletion         = "textDocument/completion"
	MethodTextDocumentHover              = "textDocument/hover"
	MethodTextDocumentSignatureHelp      = "textDocument/signatureHelp"
	MethodTextDocumentDefinition         = "textDocument/definition"
	MethodTextDocumentReferences         = "textDocument/references"
	MethodTextDocumentDocumentSymbol     = "textDocument/documentSymbol"
	MethodTextDocumentCodeAction         = "textDocument/codeAction"
	MethodTextDocumentPublishDiagnostics = "textDocument/publishDiagnostics"
	MethodWorkspaceDidChangeWatchedFiles = "workspace/didChangeWatchedFiles"
)

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type TextDocumentContentChangeEvent struct {
	Range *Range `json:"range,omitempty"`
	Text  string `json:"text"`
}

type InitializeParams struct {
	ProcessID    int                `json:"processId,omitempty"`
	RootURI      string             `json:"rootUri,omitempty"`
	Capabilities ClientCapabilities `json:"capabilities"`
	ClientInfo   ClientInfo         `json:"clientInfo,omitempty"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   ServerInfo         `json:"serverInfo,omitempty"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type InitializedParams struct{}

type ServerCapabilities struct {
	TextDocumentSync       *TextDocumentSyncOptions `json:"textDocumentSync,omitempty"`
	CompletionProvider     *CompletionOptions       `json:"completionProvider,omitempty"`
	HoverProvider          any                      `json:"hoverProvider,omitempty"`
	SignatureHelpProvider  *SignatureHelpOptions    `json:"signatureHelpProvider,omitempty"`
	DefinitionProvider     any                      `json:"definitionProvider,omitempty"`
	ReferencesProvider     any                      `json:"referencesProvider,omitempty"`
	DocumentSymbolProvider any                      `json:"documentSymbolProvider,omitempty"`
	CodeActionProvider     any                      `json:"codeActionProvider,omitempty"`
	DiagnosticsProvider    any                      `json:"diagnosticsProvider,omitempty"`
}

type TextDocumentSyncOptions struct {
	OpenClose bool         `json:"openClose,omitempty"`
	Change    int          `json:"change,omitempty"`
	Save      *SaveOptions `json:"save,omitempty"`
}

type SaveOptions struct {
	IncludeText bool `json:"includeText,omitempty"`
}

type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
	ResolveProvider   bool     `json:"resolveProvider,omitempty"`
}

type SignatureHelpOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

type CodeActionOptions struct {
	CodeActionKinds []string `json:"codeActionKinds,omitempty"`
}

type ClientCapabilities struct {
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`
	Workspace    *WorkspaceClientCapabilities    `json:"workspace,omitempty"`
}

type TextDocumentClientCapabilities struct {
	Completion         *CompletionClientCapabilities         `json:"completion,omitempty"`
	Hover              *HoverClientCapabilities              `json:"hover,omitempty"`
	SignatureHelp      *SignatureHelpClientCapabilities      `json:"signatureHelp,omitempty"`
	Definition         *DefinitionClientCapabilities         `json:"definition,omitempty"`
	References         *ReferencesClientCapabilities         `json:"references,omitempty"`
	DocumentSymbol     *DocumentSymbolClientCapabilities     `json:"documentSymbol,omitempty"`
	CodeAction         *CodeActionClientCapabilities         `json:"codeAction,omitempty"`
	PublishDiagnostics *PublishDiagnosticsClientCapabilities `json:"publishDiagnostics,omitempty"`
}

type CompletionClientCapabilities struct {
	CompletionItem *CompletionItemCapabilities `json:"completionItem,omitempty"`
}

type CompletionItemCapabilities struct {
	SnippetSupport          bool     `json:"snippetSupport,omitempty"`
	CommitCharactersSupport bool     `json:"commitCharactersSupport,omitempty"`
	DeprecatedSupport       bool     `json:"deprecatedSupport,omitempty"`
	DocumentationFormat     []string `json:"documentationFormat,omitempty"`
}

type HoverClientCapabilities struct {
	ContentFormat []string `json:"contentFormat,omitempty"`
}

type SignatureHelpClientCapabilities struct {
	SignatureInformation *SignatureInformationCapabilities `json:"signatureInformation,omitempty"`
}

type SignatureInformationCapabilities struct {
	DocumentationFormat []string `json:"documentationFormat,omitempty"`
}

type DefinitionClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

type ReferencesClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

type DocumentSymbolClientCapabilities struct {
	SymbolKind *SymbolKindCapabilities `json:"symbolKind,omitempty"`
}

type SymbolKindCapabilities struct {
	ValueSet []int `json:"valueSet,omitempty"`
}

type CodeActionClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

type PublishDiagnosticsClientCapabilities struct {
	RelatedInformation bool `json:"relatedInformation,omitempty"`
}

type WorkspaceClientCapabilities struct {
	DidChangeWatchedFiles *DidChangeWatchedFilesCapabilities `json:"didChangeWatchedFiles,omitempty"`
}

type DidChangeWatchedFilesCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

type CompletionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Context      *CompletionContext     `json:"context,omitempty"`
}

type CompletionContext struct {
	TriggerKind      int    `json:"triggerKind"`
	TriggerCharacter string `json:"triggerCharacter,omitempty"`
}

const (
	CompletionTriggerInvoked          = 0
	CompletionTriggerTriggerCharacter = 1
	CompletionTriggerIncomplete       = 2
)

type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

type CompletionItem struct {
	Label               string     `json:"label"`
	Kind                int        `json:"kind,omitempty"`
	Detail              string     `json:"detail,omitempty"`
	Documentation       any        `json:"documentation,omitempty"`
	Deprecated          bool       `json:"deprecated,omitempty"`
	Preselect           bool       `json:"preselect,omitempty"`
	SortText            string     `json:"sortText,omitempty"`
	FilterText          string     `json:"filterText,omitempty"`
	InsertText          string     `json:"insertText,omitempty"`
	InsertTextFormat    int        `json:"insertTextFormat,omitempty"`
	InsertTextMode      int        `json:"insertTextMode,omitempty"`
	TextEdit            *TextEdit  `json:"textEdit,omitempty"`
	AdditionalTextEdits []TextEdit `json:"additionalTextEdits,omitempty"`
	Command             *Command   `json:"command,omitempty"`
	Data                any        `json:"data,omitempty"`
}

const (
	InsertTextFormatPlainText = 1
	InsertTextFormatSnippet   = 2
)

const (
	CompletionItemKindText          = 1
	CompletionItemKindMethod        = 2
	CompletionItemKindFunction      = 3
	CompletionItemKindConstructor   = 4
	CompletionItemKindField         = 5
	CompletionItemKindVariable      = 6
	CompletionItemKindClass         = 7
	CompletionItemKindInterface     = 8
	CompletionItemKindModule        = 9
	CompletionItemKindProperty      = 10
	CompletionItemKindUnit          = 11
	CompletionItemKindValue         = 12
	CompletionItemKindEnum          = 13
	CompletionItemKindKeyword       = 14
	CompletionItemKindSnippet       = 15
	CompletionItemKindColor         = 16
	CompletionItemKindFile          = 17
	CompletionItemKindReference     = 18
	CompletionItemKindFolder        = 19
	CompletionItemKindEnumMember    = 20
	CompletionItemKindConstant      = 21
	CompletionItemKindStruct        = 22
	CompletionItemKindEvent         = 23
	CompletionItemKindOperator      = 24
	CompletionItemKindTypeParameter = 25
)

type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Version     int          `json:"version,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type Diagnostic struct {
	Range              Range                          `json:"range"`
	Severity           int                            `json:"severity,omitempty"`
	Code               any                            `json:"code,omitempty"`
	CodeDescription    *CodeDescription               `json:"codeDescription,omitempty"`
	Source             string                         `json:"source,omitempty"`
	Message            string                         `json:"message"`
	RelatedInformation []DiagnosticRelatedInformation `json:"relatedInformation,omitempty"`
	Tags               []int                          `json:"tags,omitempty"`
	Data               any                            `json:"data,omitempty"`
}

type CodeDescription struct {
	Href string `json:"href"`
}

type DiagnosticRelatedInformation struct {
	Location Location `json:"location"`
	Message  string   `json:"message"`
}

const (
	DiagnosticSeverityError       = 1
	DiagnosticSeverityWarning     = 2
	DiagnosticSeverityInformation = 3
	DiagnosticSeverityHint        = 4
)

const (
	DiagnosticTagUnnecessary = 1
	DiagnosticTagDeprecated  = 2
)

type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

type LocationLink struct {
	OriginSelectionRange *Range `json:"originSelectionRange,omitempty"`
	TargetURI            string `json:"targetUri"`
	TargetRange          Range  `json:"targetRange"`
	TargetSelectionRange Range  `json:"targetSelectionRange"`
}

type HoverParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type HoverResult struct {
	Contents MarkedContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

type MarkedContent struct {
	Kind     string `json:"kind"`
	Value    string `json:"value"`
	Language string `json:"language,omitempty"`
}

type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           int              `json:"kind"`
	Deprecated     bool             `json:"deprecated,omitempty"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

const (
	SymbolKindFile          = 1
	SymbolKindModule        = 2
	SymbolKindNamespace     = 3
	SymbolKindPackage       = 4
	SymbolKindClass         = 5
	SymbolKindMethod        = 6
	SymbolKindProperty      = 7
	SymbolKindField         = 8
	SymbolKindConstructor   = 9
	SymbolKindEnum          = 10
	SymbolKindInterface     = 11
	SymbolKindFunction      = 12
	SymbolKindVariable      = 13
	SymbolKindConstant      = 14
	SymbolKindString        = 15
	SymbolKindNumber        = 16
	SymbolKindBoolean       = 17
	SymbolKindArray         = 18
	SymbolKindObject        = 19
	SymbolKindKey           = 20
	SymbolKindNull          = 21
	SymbolKindEnumMember    = 22
	SymbolKindStruct        = 23
	SymbolKindEvent         = 24
	SymbolKindOperator      = 25
	SymbolKindTypeParameter = 26
)

type CodeActionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      CodeActionContext      `json:"context"`
}

type CodeActionContext struct {
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type CodeAction struct {
	Title       string       `json:"title"`
	Kind        string       `json:"kind,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
	IsPreferred bool         `json:"isPreferred,omitempty"`
	Command     *Command     `json:"command,omitempty"`
	Data        any          `json:"data,omitempty"`
}

type Command struct {
	Title     string `json:"title"`
	Command   string `json:"command"`
	Arguments []any  `json:"arguments,omitempty"`
}

type ReferenceParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Context      ReferenceContext       `json:"context"`
}

type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

type SignatureHelpParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Context      *SignatureHelpContext  `json:"context,omitempty"`
}

type SignatureHelpContext struct {
	TriggerKind         int            `json:"triggerKind"`
	TriggerCharacter    string         `json:"triggerCharacter,omitempty"`
	IsRetrigger         bool           `json:"isRetrigger,omitempty"`
	ActiveSignatureHelp *SignatureHelp `json:"activeSignatureHelp,omitempty"`
}

const (
	SignatureHelpTriggerInvoked          = 0
	SignatureHelpTriggerTriggerCharacter = 1
	SignatureHelpContentChange           = 2
)

type SignatureHelp struct {
	Signatures      []SignatureInformation `json:"signatures"`
	ActiveSignature int                    `json:"activeSignature,omitempty"`
	ActiveParameter int                    `json:"activeParameter,omitempty"`
}

type SignatureInformation struct {
	Label         string                 `json:"label"`
	Documentation any                    `json:"documentation,omitempty"`
	Parameters    []ParameterInformation `json:"parameters,omitempty"`
}

type ParameterInformation struct {
	Label         any `json:"label"`
	Documentation any `json:"documentation,omitempty"`
}

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

type DidSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Text         string                 `json:"text,omitempty"`
}

type ShutdownResult struct{}

type DidChangeWatchedFilesParams struct {
	Changes []FileEvent `json:"changes"`
}

type FileEvent struct {
	URI  string `json:"uri"`
	Type int    `json:"type"`
}

const (
	FileCreated = 1
	FileChanged = 2
	FileDeleted = 3
)

type RegistrationParams struct {
	Registrations []Registration `json:"registrations"`
}

type Registration struct {
	ID              string `json:"id"`
	Method          string `json:"method"`
	RegisterOptions any    `json:"registerOptions,omitempty"`
}

type UnregistrationParams struct {
	Unregistrations []Unregistration `json:"unregistrations"`
}

type Unregistration struct {
	ID     string `json:"id"`
	Method string `json:"method"`
}

type RequestParams struct {
	Method string `json:"method"`
}
