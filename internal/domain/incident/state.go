package incident

import (
	"errors"
	"fmt"

	"github.com/example/route-analysis-service/internal/domain/model"
)

// TransitionError describes an invalid incident state transition and is safe
// to render back to API consumers.
type TransitionError struct {
	Incident model.IncidentID
	From     model.IncidentStatus
	To       model.IncidentStatus
}

func (e *TransitionError) Error() string {
	return fmt.Sprintf("incident %s cannot move from %s to %s", e.Incident, e.From, e.To)
}

// AllowedTransitions returns the legal outgoing statuses for an incident.
func AllowedTransitions(from model.IncidentStatus) map[model.IncidentStatus]bool {
	switch from {
	case model.IncidentDraft:
		return map[model.IncidentStatus]bool{model.IncidentActive: true, model.IncidentRevoked: true}
	case model.IncidentActive:
		return map[model.IncidentStatus]bool{model.IncidentResolved: true, model.IncidentRevoked: true}
	case model.IncidentResolved:
		return map[model.IncidentStatus]bool{model.IncidentActive: true}
	case model.IncidentRevoked:
		return map[model.IncidentStatus]bool{model.IncidentDraft: true}
	default:
		return nil
	}
}

// Transition applies a single status change and returns the updated incident.
func Transition(current model.Incident, next model.IncidentStatus) (model.Incident, error) {
	allowed := AllowedTransitions(current.Status)
	if !allowed[next] {
		return current, &TransitionError{Incident: current.ID, From: current.Status, To: next}
	}
	current.Status = next
	return current, nil
}

// MustTransition is Transition without the error return, for callers that have
// already validated the target status.
func MustTransition(current model.Incident, next model.IncidentStatus) model.Incident {
	updated, err := Transition(current, next)
	if err != nil {
		panic(err)
	}
	return updated
}

// ValidateLifecycle returns an error when the incident cannot legally move to
// the target status.
func ValidateLifecycle(id model.IncidentID, from, to model.IncidentStatus) error {
	if !AllowedTransitions(from)[to] {
		return &TransitionError{Incident: id, From: from, To: to}
	}
	return nil
}

var ErrUnknownStatus = errors.New("unknown incident status")

// NormalizeStatus converts a raw string into an IncidentStatus.
func NormalizeStatus(raw string) (model.IncidentStatus, error) {
	switch model.IncidentStatus(raw) {
	case model.IncidentDraft, model.IncidentActive, model.IncidentResolved, model.IncidentRevoked:
		return model.IncidentStatus(raw), nil
	default:
		return "", ErrUnknownStatus
	}
}
