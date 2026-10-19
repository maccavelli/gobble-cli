package personality

import "testing"

// todaysText is each key's text as gobble's code held it before 0022-PLAN
// P1 moved it to the boring catalog, with its values as placeholders. The
// boring catalog must keep it word for word: it is the regression baseline
// (0022-MADR D1, D17).
var todaysText = map[Key]string{
	RetryOffline:            "waiting for the network",
	RetryEmptyReply:         "retrying: empty reply (attempt {attempt} of {max})",
	RetryAttempt:            "retrying: {cause} (attempt {attempt} of {max})",
	CompactionStart:         "compacting the conversation ({reason})",
	CompactionFailed:        "compaction failed: {error}",
	CompactionFailedTurn:    "compaction failed: {error} (automatic compaction paused for this turn)",
	CompactionFailedSession: "compaction failed: {error} (automatic compaction paused until a compaction succeeds or the model changes)",
	TurnWorking:             "working",
	TurnStopMaxTokens:       "max_tokens",
	TurnStopMaxRequests:     "max_turn_requests",
	TurnStopRefusal:         "refusal",
	TurnStopCancelled:       "cancelled",
	LinesHidden:             "{n} lines hidden (--verbose shows all)",
	CommandUnknown:          "Unknown command /{name}. /help lists the commands.",
	HelpHeading:             "Commands:",
	UsageHeading:            "Session usage",
	UsageCostUnknown:        "Cost: unknown (no price for {model} in the catalog)",
	UsageEstimated:          "These counts are estimated: the provider did not report usage.",
	UsageNoAccount:          "Account and rate-limit figures are not reported by {provider}.",
	SessionHeading:          "Session Info",
	ModelNone:               "No models are offered: no model provider is configured.",
	ModelCurrent:            "Model: {model}",
	ModelList:               "Models: {models}",
	ModelUnknown:            "Unknown model {model}. Available models: {models}.",
	ModelSet:                "Model: {model}",
	ThinkingCurrent:         "Thinking level: {level}",
	ThinkingList:            "Levels: {levels}",
	ThinkingUnknown:         "Unknown thinking level {level}. Available levels: {levels}.",
	ThinkingSet:             "Thinking level: {level}",
	ModeCurrent:             "Mode: {mode}",
	ModeList:                "Modes: {modes}",
	ModeUnknown:             "Unknown mode {mode}. The modes are {modes}.",
	ModeAlreadyPlan:         "Already in plan mode.",
	ModePlanOn:              "Mode: plan. Files cannot change and commands cannot run until the user approves a plan.",
	ModeSet:                 "Mode: {mode}",
	NameCurrent:             "Session name: {name}",
	NameUsage:               "Usage: /name <name>",
	NameNormalized:          "Session name was normalized from {from} to {to}.",
	NameSet:                 "Session name set: {name}",
	ForkNone:                "No messages to fork from yet.",
	ForkList:                "Send /fork <entry> to start a new session before one of these messages:",
	ForkNothingBefore:       "Nothing comes before that message; start a new session instead.",
	ForkNoMessage:           "No user message {id} in this session. /fork lists them.",
	ForkDone:                "Forked to session {id}.",
	CloneNone:               "Nothing to clone yet",
	CloneDone:               "Cloned to session {id}.",
	TodosNone:               "no todo list",
	CompactStart:            "Compacting the conversation…",
	CompactFailed:           "Compaction failed: {error}",
	CompactDone:             "Compacted from {tokens} tokens.",
	// New with 0022-PLAN P3: boring's text is gobble's plainest.
	PersonalityCurrent:  "Personality: {name}",
	PersonalityList:     "Personalities:",
	PersonalitySet:      "Personality: {name}",
	PersonalityUnknown:  "Unknown personality {name}. The personalities are {names}.",
	PersonalityDeclined: "Bombastic was not chosen; the personality stays {name}.",
	HumorTwo:            "There is no humor level 2. The levels are 1, 3, 4 and 5.",
	HumorUnknown:        "Unknown humor level {level}. The levels are 1, 3, 4 and 5.",
	// New with 0005-PLAN F6-1.
	TrustUntrusted:   "this project is not trusted: its .gobble files and skills are not loaded; gobble trust trusts it",
	TrustCurrent:     "This project, {root}, is {decision}.",
	TrustSaved:       "Saved: {path} is {state}. It applies from this project's next session.",
	TrustNotSaved:    "Nothing was saved.",
	TrustFailed:      "The decision was not saved: {error}",
	TrustUnavailable: "This agent cannot save a trust decision; gobble trust saves one.",
	// New with 0005-PLAN F6-2b.
	PermissionsDefaults: "With no rule, a read-only built-in tool runs, and every other call asks, an MCP tool's among them; a path outside the workspace asks.",
	PermissionsRule:     "{number}. {rule} ({from})",
	PermissionsNone:     "No rules are set.",
	PermissionsLimits: "Whatever the rules say, a write under .git or ~/.ssh, or to gobble's credential, trust or permission files, is refused; " +
		"a write to gobble's or the project's configuration, an AGENTS.md or the project's skills, and every delete, asks.",
	PermissionsRemoved: "Removed {rule} ({from}).",
	PermissionsFixed:   "Rule {number}, {rule}, is set in {from}; change it there.",
	PermissionsNoRule:  "There is no rule {number}; /permissions lists them.",
	PermissionsAdded:   "Added {rule} ({from}).",
	PermissionsBad:     "That is not a rule: {error}",
	PermissionsUsage: "/permissions lists the rules; /permissions remove N removes one; " +
		"/permissions allow|ask|deny RULE [session|project|global] adds one, to the session by default.",
	PermissionsFailed: "{rule} was not saved where you chose: {error}. It applies to this session.",
}

// The boring catalog holds today's text, word for word, for every key.
func TestBoringIsTodaysText(t *testing.T) {
	if len(todaysText) != len(Keys) {
		t.Fatalf("%d keys pinned, %d keys", len(todaysText), len(Keys))
	}
	for _, k := range Keys {
		if got := boringCat().Variants(k); len(got) != 1 || got[0] != todaysText[k] {
			t.Errorf("%s: %q, want %q", k, got, todaysText[k])
		}
	}
}
