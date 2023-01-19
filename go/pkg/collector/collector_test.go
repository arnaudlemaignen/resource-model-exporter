package collector

import (
	"strings"
	"testing"
	"resource-model-exporter/pkg/types"
	time "github.com/hyperjumptech/jiffy"

)

//collector_prom.go
func TestSubstitueVars(t *testing.T) {
	timeInterval, _ := time.DurationOf("5m")
	want := "scrape_samples_scraped{job=\"kubernetes-node-exporter\",instance=\"ip-xx-xx-x-xx.eu-west-2.compute.internal\"}"
	got := SubstitueVars("scrape_samples_scraped{job=\"kubernetes-node-exporter\",instance=\"$node\"}", []types.Var{types.Var{Name: "foo", Value: "foo"}, types.Var{Name: "node", Value: "ip-xx-xx-x-xx.eu-west-2.compute.internal"}},timeInterval)

	if !strings.Contains(got, want) {
		t.Errorf("SubstitueVars() = %q, want %q", got, want)
	}
}