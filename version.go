package main

// These variables will be set at build time using -ldflags
var (
	// VERSION Version represents the current version of the application
	VERSION = "0.0.0"
	// GITCOMMIT represents the git commit hash
	GITCOMMIT = "unknown"
	// BUILDTIME represents when the binary was built
	BUILDTIME = "unknown"
)
