package server

// Config is the wiring the server needs at startup.
type Config struct {
	DataDirs []string
	Listen   string
	Demo     bool
}
