package workflowprobe

import (
	"errors"
	"fmt"
	"io"
)

func Run(args []string, output io.Writer) error {
	if len(args) != 1 {
		return errors.New("expected exactly one workflow name")
	}
	var run func() error
	switch args[0] {
	case "plan-revision-lifecycle":
		run = runPlanRevisionLifecycle
	case "node-lease-allocation":
		run = runNodeLeaseAllocation
	case "probe-ingestion":
		run = runProbeIngestion
	case "consistency-assessment":
		run = runConsistencyAssessment
	default:
		return fmt.Errorf("unsupported workflow %q", args[0])
	}
	if err := run(); err != nil {
		return fmt.Errorf("%s: %w", args[0], err)
	}
	_, _ = fmt.Fprintf(output, "%s: passed\n", args[0])
	return nil
}

type apiFailure struct {
	Status  int
	Code    string
	Message string
}

func (e *apiFailure) Error() string {
	return fmt.Sprintf("HTTP %d %s: %s", e.Status, e.Code, e.Message)
}

func expectFailure(err error, status int, code string) error {
	var failure *apiFailure
	if !errors.As(err, &failure) {
		if err == nil {
			return fmt.Errorf("expected HTTP %d %s but request succeeded", status, code)
		}
		return fmt.Errorf("expected HTTP %d %s but got %v", status, code, err)
	}
	if failure.Status != status || failure.Code != code {
		return fmt.Errorf("expected HTTP %d %s, got HTTP %d %s", status, code, failure.Status, failure.Code)
	}
	return nil
}

func ensure(condition bool, message string) error {
	if condition {
		return nil
	}
	return errors.New(message)
}
