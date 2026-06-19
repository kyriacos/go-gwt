package gh

import (
	"encoding/json"
	"fmt"
	"strings"
)

// checkJSON mirrors a row of `gh pr checks <branch> --json bucket,state`.
//
// gh reports each check's outcome in two overlapping fields:
//   - bucket: a normalized rollup category. Observed values: "pass", "fail",
//     "pending", "skipping", "cancel".
//   - state:  the raw check state, e.g. "SUCCESS", "FAILURE", "PENDING",
//     "QUEUED", "IN_PROGRESS", "ERROR", "TIMED_OUT", "CANCELLED", "SKIPPED",
//     "NEUTRAL".
//
// We prefer bucket (it is gh's own normalization) and fall back to state when
// bucket is absent, to be tolerant of gh output variations across versions.
type checkJSON struct {
	Bucket string `json:"bucket"`
	State  string `json:"state"`
}

// outcome classifies a single check into pass / fail / pending / ignore.
type outcome int

const (
	outIgnore outcome = iota // skipped, neutral, cancelled — not counted
	outPass
	outFail
	outPending
)

func classify(bucket, state string) outcome {
	switch strings.ToLower(strings.TrimSpace(bucket)) {
	case "pass":
		return outPass
	case "fail":
		return outFail
	case "pending":
		return outPending
	case "skipping", "cancel":
		return outIgnore
	case "":
		// fall through to state-based classification
	default:
		return outIgnore
	}

	switch strings.ToUpper(strings.TrimSpace(state)) {
	case "SUCCESS":
		return outPass
	case "FAILURE", "ERROR", "TIMED_OUT", "ACTION_REQUIRED", "STARTUP_FAILURE":
		return outFail
	case "PENDING", "QUEUED", "IN_PROGRESS", "WAITING", "REQUESTED", "EXPECTED":
		return outPending
	default:
		// SKIPPED, NEUTRAL, CANCELLED, STALE, or unknown — do not count.
		return outIgnore
	}
}

// Checks rolls up the CI checks for the given branch into a CIStatus.
//
// Rollup rules:
//   - Failed > 0          -> CIFailing
//   - any pending/queued  -> CIPending
//   - Total > 0 (all pass)-> CIPassing
//   - no checks at all     -> CIUnknown
//
// Total counts only pass + fail + pending; ignored checks (skipped, neutral,
// cancelled) do not contribute. Returns ErrUnavailable when gh is missing or
// unauthenticated.
func (c *CmdClient) Checks(branch string) (CIStatus, error) {
	if !c.Available() {
		return CIStatus{}, ErrUnavailable
	}

	// gh pr checks exits non-zero when checks are failing or still pending, yet
	// still emits valid JSON on stdout. We therefore call the runner directly
	// (rather than via runGH) so we can inspect stdout regardless of exit code,
	// and only surface the exec error when the output is not parseable.
	stdout, stderr, runErr := c.run.Run(c.ctx, "", "gh", "pr", "checks", branch, "--json", "bucket,state")
	if status, ok := parseChecks(stdout); ok {
		return status, nil
	}
	if runErr != nil {
		msg := strings.TrimSpace(string(stderr))
		if msg != "" {
			return CIStatus{}, fmt.Errorf("gh pr checks %s: %w: %s", branch, runErr, msg)
		}
		return CIStatus{}, fmt.Errorf("gh pr checks %s: %w", branch, runErr)
	}
	return CIStatus{}, fmt.Errorf("gh pr checks %s: parse json: unexpected output", branch)
}

// parseChecks parses the JSON array from `gh pr checks --json` and returns the
// rolled-up status. ok is false when the payload is not valid JSON.
func parseChecks(out []byte) (CIStatus, bool) {
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		// No output: treat as no checks rather than a parse failure.
		return CIStatus{State: CIUnknown}, true
	}
	var rows []checkJSON
	if err := json.Unmarshal([]byte(trimmed), &rows); err != nil {
		return CIStatus{}, false
	}

	var status CIStatus
	pending := 0
	for _, r := range rows {
		switch classify(r.Bucket, r.State) {
		case outPass:
			status.Passed++
			status.Total++
		case outFail:
			status.Failed++
			status.Total++
		case outPending:
			pending++
			status.Total++
		case outIgnore:
			// not counted
		}
	}

	switch {
	case status.Failed > 0:
		status.State = CIFailing
	case pending > 0:
		status.State = CIPending
	case status.Total > 0:
		status.State = CIPassing
	default:
		status.State = CIUnknown
	}
	return status, true
}
