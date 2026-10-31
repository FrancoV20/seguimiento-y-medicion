package defect

import "fmt"

// Resolved es el resultado de resolver un defecto.
type Resolved struct {
	DefectID     int64
	SprintNumber int64
}

// CheckResolve aplica las reglas para resolver un defecto. El estado se comprueba primero: un
// defecto ya resuelto informa ErrAlreadyResolved aunque no haya Sprint activo. Después se exige un
// Sprint activo cuyo número no sea anterior al del Sprint de detección.
func CheckResolve(status Status, detectionSprintNumber, activeSprintNumber int64, hasActiveSprint bool) error {
	switch status {
	case StatusOpen:
	case StatusResolved:
		return ErrAlreadyResolved
	default:
		return fmt.Errorf("%w: %q", errUnknownDefectStatus, status)
	}
	if !hasActiveSprint {
		return ErrNoActiveSprint
	}
	if activeSprintNumber < detectionSprintNumber {
		return ErrResolutionBeforeDetection
	}
	return nil
}

// CheckReopen aplica la regla para reabrir un defecto: solo se reabre uno "Resuelto". Reabrir no
// necesita Sprint activo.
func CheckReopen(status Status) error {
	switch status {
	case StatusResolved:
		return nil
	case StatusOpen:
		return ErrAlreadyOpen
	default:
		return fmt.Errorf("%w: %q", errUnknownDefectStatus, status)
	}
}
