package utils

import (
	"os"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

// //api_prom.go
// func TestProm_query(t *testing.T) {
// 	//up{instance="demo.robustperception.io:9090", job="prometheus"} => 1 @[1603810141.965]
// 	want := "up{instance=\"demo.do.prometheus.io:9090\", job=\"prometheus\"} => 1"
// 	result, _ := Prom_query("http://demo.robustperception.io:9090", "up{job=\"prometheus\"}")
// 	got := result.String()

// 	if !strings.Contains(got, want) {
// 		t.Errorf("TestProm_query() = %q, want %q", got, want)
// 	}
// }

func TestPromNoConnection_query(t *testing.T) {
	want := "server misbehaving"
	want2 := "no such host"
	_, err := Prom_query("http://foo:9090", "up{job=\"prometheus\"}")
	log.Info(err)
	if !strings.Contains(err.Error(), want) && !strings.Contains(err.Error(), want2) {
		t.Errorf("TestPromNoConnection_query() = %q, want %q or %q", err, want, want2)
	}
}

// func TestProm_queryRange(t *testing.T) {
// 	//{instance=\"demo.robustperception.io:9090\", job=\"prometheus\"} =>\n1589.5172413793105 @[1603808650.416]
// 	want := "{instance=\"demo.do.prometheus.io:9090\", job=\"prometheus\"} =>"
// 	result, _ := Prom_queryRange("http://demo.robustperception.io:9090", "rate(prometheus_tsdb_head_samples_appended_total[5m])", time.Now().Add(-time.Hour), time.Now(), time.Minute)
// 	got := result.String()

// 	if !strings.Contains(got, want) {
// 		t.Errorf("Prom_queryRange() = %q, want %q", got, want)
// 	}
// }

func TestPromNoConnection_queryRange(t *testing.T) {
	want := "server misbehaving"
	want2 := "no such host"
	_, err := Prom_queryRange("http://foo:9090", "up{job=\"prometheus\"}", time.Now().Add(-time.Hour), time.Now(), time.Minute)
	log.Info(err)
	if !strings.Contains(err.Error(), want) {
		t.Errorf("TestPromNoConnection_queryRange() = %q, want %q or %q", err, want, want2)
	}
}

// func TestProm_series(t *testing.T) {
// 	//{__name__=\"scrape_duration_seconds\", instance=\"demo.robustperception.io:9090\", job=\"prometheus\"} {__name__=\"scrape_samp
// 	want := "{__name__=\"scrape_duration_seconds\", instance=\"demo.do.prometheus.io:9090\", job=\"prometheus\"}"
// 	got, _ := Prom_series("http://demo.robustperception.io:9090", "{__name__=~\"scrape_.+\",job=\"prometheus\"}", time.Now().Add(-time.Hour), time.Now())

// 	if !strings.Contains(got, want) {
// 		t.Errorf("Prom_series() = %q, want %q", got, want)
// 	}
// }

func TestPromNoConnection_series(t *testing.T) {
	want := "server misbehaving"
	want2 := "no such host"
	_, err := Prom_series("http://foo:9090", "up{job=\"prometheus\"}", time.Now().Add(-time.Hour), time.Now())
	log.Info(err)
	if !strings.Contains(err.Error(), want) {
		t.Errorf("TestPromNoConnection_series() = %q, want %q or %q", err, want, want2)
	}
}

func TestEnv(t *testing.T) {

	os.Setenv("SERVER_HOST", "test")
	os.Setenv("COMPLEXITY_LIMIT", "10")
	os.Setenv("PLAYGROUND_ENABLED", "true")

	want := "test"
	got := GetStringEnv("SERVER_HOST", "else")

	if got != want {
		t.Errorf("TestEnv = %q, want %q", got, want)
	}

	wantInt := 10
	gotInt := GetIntEnv("COMPLEXITY_LIMIT", 5)

	if gotInt != wantInt {
		t.Errorf("TestEnv = %d, want %d", gotInt, wantInt)
	}

	wantInt64 := int64(10)
	gotInt64 := GetInt64Env("COMPLEXITY_LIMIT", int64(5))

	if gotInt != wantInt {
		t.Errorf("TestEnv = %d, want %d", gotInt64, wantInt64)
	}

	wantBool := true
	gotBool := GetBoolEnv("PLAYGROUND_ENABLED", false)

	if gotBool != wantBool {
		t.Errorf("TestEnv = %t, want %t", gotBool, wantBool)
	}
}
