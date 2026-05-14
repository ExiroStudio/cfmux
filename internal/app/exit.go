package app

import "fmt"

// ExitError carries a desired process exit code up to the CLI entrypoint.
// Library code should never call os.Exit directly — return an ExitError
// instead and let cmd.Execute translate it into a process exit.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("exit %d", e.Code)
}

func (e *ExitError) Unwrap() error {
	return e.Err
}
