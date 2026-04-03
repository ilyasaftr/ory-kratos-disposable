package domain

type FailureMode string

const (
	FailureModeOpen   FailureMode = "open"
	FailureModeClosed FailureMode = "closed"
)

func (m FailureMode) String() string {
	return string(m)
}
