package bootstrap

// GlobalRegistry is the process-wide feature registry. Features register themselves
// against it in their package init() functions, so simply importing a feature package
// makes it available to the CLI.
var GlobalRegistry = NewRegistry()
