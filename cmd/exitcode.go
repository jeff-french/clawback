package cmd

// ExitError signals a clean exit with a specific non-zero code.
// Commands that use distinct exit codes to communicate status (e.g. diff:
// 0 = clean, 1 = drifted, 2 = error) return this instead of calling os.Exit,
// so deferred cleanup runs and tests can intercept the value.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string { return "" }
