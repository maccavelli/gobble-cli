package acpext

// The requests gobble's agent serves, which initialize advertises
// (0002-PLAN Phase 7's frozen table, 0005-PLAN F5-1 and F5-2, and
// 0022-MADR D6). Each one's params and result are in the package doc.
const (
	// MethodSteer queues a message for the running turn to take at its next
	// step.
	MethodSteer = "_gobble/steer"
	// MethodFollowUp queues a message for after the running turn.
	MethodFollowUp = "_gobble/follow_up"
	// MethodClearQueue empties both queues and returns what they held.
	MethodClearQueue = "_gobble/clear_queue"
	// MethodSetSteeringMode sets how the steering queue is taken.
	MethodSetSteeringMode = "_gobble/set_steering_mode"
	// MethodSetFollowUpMode sets how the follow-up queue is taken.
	MethodSetFollowUpMode = "_gobble/set_follow_up_mode"
	// MethodCompact compacts the session's context now. It holds the
	// session's turn.
	MethodCompact = "_gobble/compact"
	// MethodUsage is /usage's text, with the figures it is made from.
	MethodUsage = "_gobble/usage"
	// MethodSetSessionName names the session, as /name does. It holds the
	// session's turn.
	MethodSetSessionName = "_gobble/set_session_name"
	// MethodFork writes a new session that ends before a user message, for
	// the client to load. It holds the session's turn.
	MethodFork = "_gobble/fork"
	// MethodClone writes a new session holding the whole path, for the
	// client to load. It holds the session's turn.
	MethodClone = "_gobble/clone"
	// MethodGetState is the session's state.
	MethodGetState = "_gobble/get_state"
	// MethodGetMessages are the messages the model sees.
	MethodGetMessages = "_gobble/get_messages"
	// MethodGetEntries are the session file's entries, or those after one.
	MethodGetEntries = "_gobble/get_entries"
	// MethodGetLastAssistantText is the last assistant message's text.
	MethodGetLastAssistantText = "_gobble/get_last_assistant_text"
	// MethodGetSessionStats are the session's message and token counts.
	MethodGetSessionStats = "_gobble/get_session_stats"
	// MethodGetForkMessages are the user messages a fork may end before.
	MethodGetForkMessages = "_gobble/get_fork_messages"
	// MethodSetAutoRetry turns the turn's own retries on or off.
	MethodSetAutoRetry = "_gobble/set_auto_retry"
	// MethodAbortRetry ends the running turn's retry wait.
	MethodAbortRetry = "_gobble/abort_retry"
	// MethodSetAutoCompaction turns automatic compaction on or off.
	MethodSetAutoCompaction = "_gobble/set_auto_compaction"
	// MethodSetPersonality sets the session's personality.
	MethodSetPersonality = "_gobble/set_personality"
)

// The notifications gobble's agent sends.
const (
	// MethodRetry is a turn's retry wait, at its start and its end
	// (0005-PLAN F5-1 decision 5). Its params are a RetryNotice.
	MethodRetry = "_gobble/retry"
	// MethodCompaction is a compaction's start and end (0005-PLAN F5-2
	// decision 9). Its params are a CompactionNotice.
	MethodCompaction = "_gobble/compaction"
	// MethodPersonality is a session's personality, at its start and at
	// each change (0022-MADR D6). Its params are a PersonalityNotice.
	MethodPersonality = "_gobble/personality"
)

// Requests returns the requests gobble's agent serves, sorted: the list
// initialize advertises in CapabilitiesMeta. Each call returns a new slice.
func Requests() []string {
	return []string{
		MethodAbortRetry,
		MethodClearQueue,
		MethodClone,
		MethodCompact,
		MethodFollowUp,
		MethodFork,
		MethodGetEntries,
		MethodGetForkMessages,
		MethodGetLastAssistantText,
		MethodGetMessages,
		MethodGetSessionStats,
		MethodGetState,
		MethodSetAutoCompaction,
		MethodSetAutoRetry,
		MethodSetFollowUpMode,
		MethodSetPersonality,
		MethodSetSessionName,
		MethodSetSteeringMode,
		MethodSteer,
		MethodUsage,
	}
}

// Notifications returns the notifications gobble's agent sends, sorted.
// Each call returns a new slice.
func Notifications() []string {
	return []string{MethodCompaction, MethodPersonality, MethodRetry}
}
