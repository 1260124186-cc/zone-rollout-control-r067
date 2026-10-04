package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"time"
)

func ActivatePlan(plan *RolloutPlan, now time.Time) error {
	if plan.State != PlanDraft {
		return InvalidState("plan can be activated only from draft", map[string]any{
			"plan_id": plan.ID,
			"state":   plan.State,
		})
	}
	if plan.LatestRevisionID == "" {
		return InvalidState("plan needs a zone revision before activation", map[string]any{
			"plan_id": plan.ID,
		})
	}
	plan.State = PlanActive
	plan.ActivatedAt = timePointer(now.UTC())
	plan.Version++
	return nil
}

func CompletePlan(plan *RolloutPlan, now time.Time) error {
	if plan.State != PlanActive {
		return InvalidState("plan can be completed only from active", map[string]any{
			"plan_id": plan.ID,
			"state":   plan.State,
		})
	}
	plan.State = PlanCompleted
	plan.CompletedAt = timePointer(now.UTC())
	plan.Version++
	return nil
}

func LeaseFingerprint(planID string, input LeaseInput) string {
	payload := fmt.Sprintf(
		"%s|%s|%s|%d|%d",
		planID,
		input.RequestKey,
		input.Region,
		input.LeaseSeconds,
		input.MaxNodes,
	)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func ProbeFingerprint(leaseID string, input ProbeInput) string {
	payload := fmt.Sprintf(
		"%s|%s|%t|%d|%d|%s",
		leaseID,
		input.RequestKey,
		input.Success,
		input.ObservedSerial,
		input.LatencyMS,
		input.ObservedAt.UTC().Format(time.RFC3339Nano),
	)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func BuildAssessment(
	id string,
	plan RolloutPlan,
	revision ZoneRevision,
	probes []Probe,
	now time.Time,
) RolloutAssessment {
	reasons := make(map[string]struct{})
	if len(probes) == 0 {
		reasons["no_probes"] = struct{}{}
	}
	probeCount := len(probes)
	successCount := 0
	maximumLatency := 0
	nodes := make(map[string]struct{})
	for _, probe := range probes {
		nodes[probe.NodeID] = struct{}{}
		if probe.LatencyMS > maximumLatency {
			maximumLatency = probe.LatencyMS
		}
		if probe.Success {
			successCount++
			if probe.ObservedSerial != revision.Serial {
				reasons["serial_mismatch"] = struct{}{}
			}
		}
	}
	if len(nodes) < plan.ExpectedNodes {
		reasons["node_not_observed"] = struct{}{}
	}
	successRatio := 0.0
	if probeCount > 0 {
		successRatio = round(float64(successCount)/float64(probeCount), 6)
	}
	if successRatio < plan.MinimumSuccessRatio {
		reasons["success_ratio_below_minimum"] = struct{}{}
	}
	if maximumLatency > plan.MaximumLatencyMS {
		reasons["latency_limit_exceeded"] = struct{}{}
	}
	reasonList := make([]string, 0, len(reasons))
	for reason := range reasons {
		reasonList = append(reasonList, reason)
	}
	sort.Strings(reasonList)
	verdict := VerdictConsistent
	if len(reasonList) > 0 {
		verdict = VerdictReviewRequired
	}
	return RolloutAssessment{
		ID:               id,
		PlanID:           plan.ID,
		RevisionID:       revision.ID,
		Verdict:          verdict,
		Reasons:          reasonList,
		ProbeCount:       probeCount,
		SuccessCount:     successCount,
		SuccessRatio:     successRatio,
		MaximumLatencyMS: maximumLatency,
		CreatedAt:        now.UTC(),
	}
}

func round(value float64, places int) float64 {
	scale := math.Pow10(places)
	return math.Round(value*scale) / scale
}

// NodeTimelineFor resolves the recorded availability timeline for a node.
func NodeTimelineFor(state *State, node ResolverNode) (NodeTimeline, bool) {
	if state == nil {
		return NodeTimeline{}, false
	}
	timeline, ok := state.NodeTimelines[node.ID]
	return timeline, ok
}

// NodeTimelineChanged reports whether two recorded timelines describe a
// different availability instant or version.
func NodeTimelineChanged(recorded NodeTimeline, candidate NodeTimeline) bool {
	if !recorded.UpdatedAt.Equal(candidate.UpdatedAt) {
		return true
	}
	return recorded.Version != candidate.Version
}

// RecordNodeTimeline stores the availability time and version a node has
// reached so later reads observe the same recency without touching the
// aggregate. The registry only moves forward: a stale observation never pulls
// a node's recorded business time back to an earlier instant.
func RecordNodeTimeline(state *State, node ResolverNode) NodeTimeline {
	candidate := NodeTimeline{UpdatedAt: node.UpdatedAt.UTC(), Version: node.Version}
	if state == nil {
		return candidate
	}
	if state.NodeTimelines == nil {
		state.NodeTimelines = make(map[string]NodeTimeline)
	}
	recorded, ok := state.NodeTimelines[node.ID]
	if !ok {
		state.NodeTimelines[node.ID] = candidate
		return candidate
	}
	merged := recorded
	if candidate.UpdatedAt.After(merged.UpdatedAt) {
		merged.UpdatedAt = candidate.UpdatedAt
	}
	if candidate.Version > merged.Version {
		merged.Version = candidate.Version
	}
	if !NodeTimelineChanged(recorded, merged) {
		return recorded
	}
	state.NodeTimelines[node.ID] = merged
	return merged
}

// EnsureNodeTimeline seeds a timeline for a node that has not been observed
// yet so list reads always resolve the same registry shape.
func EnsureNodeTimeline(state *State, node ResolverNode) NodeTimeline {
	if timeline, ok := NodeTimelineFor(state, node); ok {
		return timeline
	}
	return RecordNodeTimeline(state, node)
}

// ApplyNodeTimeline projects the recorded timeline back onto a node for list
// and detail responses.
func ApplyNodeTimeline(state *State, node ResolverNode) ResolverNode {
	timeline, ok := NodeTimelineFor(state, node)
	if !ok {
		return node
	}
	node.UpdatedAt = timeline.UpdatedAt
	node.Version = timeline.Version
	return node
}

// ProjectNodeTimelines applies the recorded registry to a batch of nodes.
func ProjectNodeTimelines(state *State, nodes []ResolverNode) []ResolverNode {
	projected := make([]ResolverNode, 0, len(nodes))
	for _, node := range nodes {
		projected = append(projected, ApplyNodeTimeline(state, node))
	}
	return projected
}

// RebuildNodeTimelines restores the node timeline registry whenever a snapshot
// is loaded. The registry persisted with the snapshot is authoritative so the
// recorded business time and version survive a restart; it is only repaired
// forward from each node aggregate's current time and version (for example when
// an older snapshot lacks an entry) and never reset to the enrollment baseline.
func RebuildNodeTimelines(state *State) {
	if state == nil {
		return
	}
	if state.NodeTimelines == nil {
		state.NodeTimelines = make(map[string]NodeTimeline, len(state.Nodes))
	}
	timelines := state.NodeTimelines
	for id, node := range state.Nodes {
		recorded, ok := timelines[id]
		if !ok {
			timelines[id] = NodeTimeline{UpdatedAt: node.UpdatedAt.UTC(), Version: node.Version}
			continue
		}
		merged := recorded
		if node.UpdatedAt.UTC().After(merged.UpdatedAt) {
			merged.UpdatedAt = node.UpdatedAt.UTC()
		}
		if node.Version > merged.Version {
			merged.Version = node.Version
		}
		timelines[id] = merged
	}
	for id := range timelines {
		if _, ok := state.Nodes[id]; !ok {
			delete(timelines, id)
		}
	}
	state.NodeTimelines = timelines
}

func timePointer(value time.Time) *time.Time {
	return &value
}
