package scenario

import "strings"

const (
	ImplementationStatusReady    = "ready"
	ImplementationStatusDraft    = "draft"
	ImplementationStatusScaffold = "scaffold"
)

func NormalizeImplementationStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return ImplementationStatusReady
	}
	return status
}

func IsScaffold(exercise *Exercise) bool {
	if exercise == nil {
		return false
	}
	return NormalizeImplementationStatus(exercise.Spec.ImplementationStatus) == ImplementationStatusScaffold
}
