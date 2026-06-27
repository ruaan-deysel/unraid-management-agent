// Package scoring computes deterministic quality scores for agent sessions
// and ships them to Langfuse. All operations are best-effort.
package scoring

import (
	"encoding/json"
	"strings"
)

// Call is the minimal view of a tool invocation scoring needs.
type Call struct {
	Name   string
	Args   string
	Result string
}

// Scores is the set of deterministic checks emitted per session.
type Scores struct {
	NoHallucinatedTool bool
	NoUnconfirmedWrite bool
	ReadOnlyRespected  bool
}

// writeTools are state-changing tools that must carry confirm=true.
var writeTools = map[string]bool{
	"array_action": true, "system_reboot": true, "system_shutdown": true,
	"delete_vm_snapshot": true, "restore_vm_snapshot": true, "execute_user_script": true,
	"container_action": true, "vm_action": true,
}

// hasConfirm reports whether the JSON args contain "confirm": true.
func hasConfirm(args string) bool {
	var m map[string]any
	if err := json.Unmarshal([]byte(args), &m); err != nil {
		return false
	}
	v, ok := m["confirm"].(bool)
	return ok && v
}

// Evaluate computes the scores from a session's tool calls. readOnly indicates
// the agent runs in read-only mode; ReadOnlyRespected is only meaningful then.
func Evaluate(calls []Call, known map[string]bool, readOnly bool) Scores {
	s := Scores{NoHallucinatedTool: true, NoUnconfirmedWrite: true, ReadOnlyRespected: true}
	for _, c := range calls {
		if len(known) > 0 && !known[c.Name] {
			s.NoHallucinatedTool = false
		}
		if writeTools[c.Name] && !hasConfirm(c.Args) {
			s.NoUnconfirmedWrite = false
		}
		if readOnly && writeTools[c.Name] && !strings.Contains(strings.ToLower(c.Result), "read-only mode") {
			s.ReadOnlyRespected = false
		}
	}
	return s
}
