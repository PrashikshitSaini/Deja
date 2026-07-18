package model

// Event is one command execution stored locally by Deja.
type Event struct {
	ID         string  `json:"id"`
	Command    string  `json:"command"`
	Family     string  `json:"family"`
	CWD        string  `json:"cwd,omitempty"`
	ExitStatus *int    `json:"exit_status,omitempty"`
	OccurredAt float64 `json:"occurred_at,omitempty"`
	Duration   float64 `json:"duration,omitempty"`
	Source     string  `json:"source"`
}

// Candidate is a distinct command variant aggregated from execution events.
type Candidate struct {
	Command     string  `json:"command"`
	Display     string  `json:"display,omitempty"`
	Family      string  `json:"family"`
	Uses        int     `json:"uses"`
	LastRun     float64 `json:"last_run,omitempty"`
	Successes   int     `json:"successes"`
	StatusCount int     `json:"status_count"`
	CWDHits     int     `json:"cwd_hits"`
	Score       float64 `json:"score"`
}

// DisplayCommand returns the safe, human-facing form of a candidate. Command
// remains the exact text that Tab inserts into the editable prompt.
func (candidate Candidate) DisplayCommand() string {
	if candidate.Display != "" {
		return candidate.Display
	}
	return candidate.Command
}

func (candidate Candidate) SuccessRate() (float64, bool) {
	if candidate.StatusCount == 0 {
		return 0, false
	}
	return float64(candidate.Successes) / float64(candidate.StatusCount), true
}
