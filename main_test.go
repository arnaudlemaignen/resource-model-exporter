package main

import (
	"os"
	"strings"
	"testing"
	"time"

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

func TestPromNoConnection_query(t *testing.T) {
	want := "dial tcp: lookup foo on"
	_, err := Prom_query("http://foo:9090", "up{job=\"prometheus\"}")
	log.Info(err)
	if !strings.Contains(err.Error(), want) {
		t.Errorf("TestPromNoConnection_query() = %q, want %q", err, want)
	}
}

func TestPromNoConnection_queryRange(t *testing.T) {
	want := "dial tcp: lookup foo on"
	_, err := Prom_queryRange("http://foo:9090", "up{job=\"prometheus\"}", time.Now().Add(-time.Hour), time.Now(), time.Minute)
	log.Info(err)
	if !strings.Contains(err.Error(), want) {
		t.Errorf("TestPromNoConnection_queryRange() = %q, want %q", err, want)
	}
}

func TestPromNoConnection_series(t *testing.T) {
	want := "dial tcp: lookup foo on"
	_, err := Prom_series("http://foo:9090", "up{job=\"prometheus\"}", time.Now().Add(-time.Hour), time.Now())
	log.Info(err)
	if !strings.Contains(err.Error(), want) {
		t.Errorf("TestPromNoConnection_series() = %q, want %q", err, want)
	}
}

//collector.go
//not so much to test

//collector_prom.go
func TestSubstitueVars(t *testing.T) {
	want := "scrape_samples_scraped{job=\"kubernetes-node-exporter\",instance=\"ip-xx-xx-x-xx.eu-west-2.compute.internal\"}"
	got := SubstitueVars("scrape_samples_scraped{job=\"kubernetes-node-exporter\",instance=\"$node\"}", []Var{Var{Name: "foo", Value: "foo"}, Var{Name: "node", Value: "ip-xx-xx-x-xx.eu-west-2.compute.internal"}})

	if !strings.Contains(got, want) {
		t.Errorf("SubstitueVars() = %q, want %q", got, want)
	}
}
