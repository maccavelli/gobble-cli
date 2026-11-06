// Package acpext is gobble's ACP extension contract: the names and shapes of
// the _gobble/* requests and notifications gobble's agent serves, the
// _meta.gobble member of its initialize answer, and the auth providers and
// error data a client reads. gobble's own server and client use these types,
// and so may any other client (gobble-cli
// docs/decisions/0042-MADR-acpext-extension-contract-module.md).
//
// The package holds names, types and fixed lists, and no behaviour: no
// connection, handler, encoder or reader. It imports only the standard
// library, so a client gets it and nothing more.
//
// # Calling
//
// A client sends a request with its ACP connection's extension call and
// decodes the result into the method's type. With acp-go-sdk2:
//
//	raw, err := conn.CallExtension(ctx, acpext.MethodClone, acpext.SessionParams{SessionID: id})
//	if err != nil {
//		return err
//	}
//	var res acpext.CloneResult
//	if err := json.Unmarshal(raw, &res); err != nil {
//		return err
//	}
//
// A notification is decoded the same way, into RetryNotice,
// CompactionNotice or PersonalityNotice. The extensions gobble serves are
// listed in initialize's agentCapabilities._meta under MetaKey, which
// acp-go-sdk2 reads with acp.DecodeMeta[acpext.CapabilitiesMeta](meta,
// acpext.MetaKey).
//
// A refused request's JSON-RPC error carries ErrorData as its data. Its
// code is the protocol's: -32602 for bad params or an unknown session,
// -32600 for a writer refused while a prompt runs, and -32000 when a
// provider has no credential.
//
// # Methods
//
// Every request carries the session's id. Its params and result:
//
//	MethodSteer                 MessageParams         QueuedResult
//	MethodFollowUp              MessageParams         QueuedResult
//	MethodClearQueue            SessionParams         ClearedResult
//	MethodSetSteeringMode       ModeParams            {}
//	MethodSetFollowUpMode       ModeParams            {}
//	MethodCompact               CompactParams         CompactResult
//	MethodUsage                 SessionParams         UsageResult
//	MethodSetSessionName        SetSessionNameParams  {}
//	MethodFork                  ForkParams            ForkResult
//	MethodClone                 SessionParams         CloneResult
//	MethodGetState              SessionParams         StateResult
//	MethodGetMessages           SessionParams         MessagesResult
//	MethodGetEntries            GetEntriesParams      EntriesResult
//	MethodGetLastAssistantText  SessionParams         TextResult
//	MethodGetSessionStats       SessionParams         SessionStats
//	MethodGetForkMessages       SessionParams         ForkMessagesResult
//	MethodSetAutoRetry          EnabledParams         {}
//	MethodAbortRetry            SessionParams         {}
//	MethodSetAutoCompaction     EnabledParams         {}
//	MethodSetPersonality        SetPersonalityParams  PersonalityResult
//
// A result of {} has no type here: the call's result is an empty object.
//
// The notifications, from the agent: MethodRetry with RetryNotice,
// MethodCompaction with CompactionNotice, and MethodPersonality with
// PersonalityNotice. A session's first _gobble/personality comes before
// session/new's answer, so a client delivers it to the session it is
// opening.
//
// # Encoding
//
// The types carry JSON tags only, which encoding/json and
// encoding/json/v2 read alike. A field with omitzero is left out when it is
// its zero value; a pointer field without it is sent as null.
//
// Three fields hold gobble's session-file shapes as raw JSON, which this
// package does not freeze: CompactResult.Details, and the elements of
// MessagesResult.Messages and EntriesResult.Entries.
//
// # Versions
//
// The module is tagged acpext/vX.Y.Z. It stays at v0 until gobble's v1.
// Each change to the contract is a new minor tag, so a client should build
// these types with field names, and should take a member or a value it does
// not know, such as a new RetryCause.
package acpext
