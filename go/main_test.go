package main

import (
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
)

//Unit tests for the "main" package
var (
//
)

//Main of the test to init global vars, prepare things
func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	log.SetLevel(log.DebugLevel)
	os.Exit(m.Run())
}

//main.go
func TestReady(t *testing.T) {
	want := "Resource Model Exporter is ready"
	if got := Ready(); got != want {
		t.Errorf("Ready() = %q, want %q", got, want)
	}
}
