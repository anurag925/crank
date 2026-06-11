package bootstrap

// GlobalToolRegistry is the process-wide tool registry. Tool packages register
// themselves against it in their package init() functions, so simply importing a
// tool package makes it available to the CLI.
var GlobalToolRegistry = NewToolRegistry()
