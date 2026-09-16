package pipeline

// Handler is the core abstraction for pipeline extensions. Each
// handler declares the phase it belongs to and provides a Handle
// method that receives a mutable context. Handlers are executed in
// registration order within a phase; each handler's output becomes
// the next handler's input.
//
// A handler that returns a non-nil error aborts the chain — no
// further handlers in the same phase (or subsequent phases) will
// run.
type Handler interface {
	// Name returns a short, unique identifier for the handler
	// (e.g. "sticky", "alias", "date-normalise"). Used in
	// correction records and log output.
	Name() string

	// Phase returns the pipeline phase this handler belongs to.
	Phase() Phase

	// Handle processes the context and returns an error to abort
	// the chain, or nil to continue.
	Handle(ctx *Context) error
}
