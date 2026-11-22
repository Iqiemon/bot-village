package server

import "botvillage/ingest"

type Config struct {
	DataDirs []string
	Listen   string
	Demo     bool
	Events   <-chan ingest.Event
}
