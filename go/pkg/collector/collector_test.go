package collector

import (
	"resource-model-exporter/pkg/types"
	"strings"
	"testing"

	time "github.com/hyperjumptech/jiffy"
)

//collector_prom.go
func TestSubstitueVars(t *testing.T) {
	timeInterval, _ := time.DurationOf("5m")
	want := "scrape_samples_scraped{job=\"kubernetes-node-exporter\",instance=\"ip-xx-xx-x-xx.eu-west-2.compute.internal\"}"
	got := SubstitueVars("scrape_samples_scraped{job=\"kubernetes-node-exporter\",instance=\"$node\"}", []types.Var{{Name: "foo", Value: "foo"}, {Name: "node", Value: "ip-xx-xx-x-xx.eu-west-2.compute.internal"}}, timeInterval)

	if !strings.Contains(got, want) {
		t.Errorf("SubstitueVars() = %q, want %q", got, want)
	}
}

func TestSubstitueVars2(t *testing.T) {
	timeInterval, _ := time.DurationOf("5m")
	want := "scrape_samples_scraped{job=\"kubernetes-node-exporter\",instance=\"ip-xx-xx-x-xx.eu-west-2.compute.internal\"}"
	got := SubstitueVars("scrape_samples_scraped{job=\"kubernetes-node-exporter\",instance=\"$node\"}", []types.Var{{Name: "foo", Value: "foo"}}, timeInterval)

	//fail on purpose because $node is not provided
	if strings.Contains(got, want) {
		t.Errorf("SubstitueVars() = %q, want %q", got, want)
	}
}

func TestSubstitueVars3(t *testing.T) {
	timeInterval, _ := time.DurationOf("5m")
	want := "scrape_samples_scraped{job=\"kubernetes-node-exporter\",instance=\"ip-xx-xx-x-xx.eu-west-2.compute.internal\"}"
	got := SubstitueVars("scrape_samples_scraped{job=\"kubernetes-node-exporter\",instance=\"$1\"}", []types.Var{{Name: "foo", Value: "foo"}}, timeInterval)

	//fail on purpose because $1 should not be replaced
	if strings.Contains(got, want) {
		t.Errorf("SubstitueVars() = %q, want %q", got, want)
	}
}

/* func TestGetPredCoeffs(t *testing.T) {
	mapPred := make(map[string]string)
	mapPred["image_pixels_M"] = "predictor_0"

	wantPredName := "predictor_0"
	wantPredAlias := "image_pixels_M"
	wantDegree := "1"
	predName, predAlias, degree := getPredCoeffs("image_pixels_M", mapPred)
	if predName != wantPredName {
		t.Errorf("TestGetPredCoeffs(predName) = %q, want %q", predName, wantPredName)
	}
	if predAlias != wantPredAlias {
		t.Errorf("TestGetPredCoeffs(predAlias) = %q, want %q", predAlias, wantPredAlias)
	}
	if degree != wantDegree {
		t.Errorf("TestGetPredCoeffs(predName) = %q, want %q", degree, wantDegree)
	}

	wantDegree = "2"
	predName, predAlias, degree = getPredCoeffs("(image_pixels_M)^2", mapPred)
	if predName != wantPredName {
		t.Errorf("TestGetPredCoeffs(predName 2) = %q, want %q", predName, wantPredName)
	}
	if predAlias != wantPredAlias {
		t.Errorf("TestGetPredCoeffs(predAlias 2) = %q, want %q", predAlias, wantPredAlias)
	}
	if degree != wantDegree {
		t.Errorf("TestGetPredCoeffs(predName 2) = %q, want %q", degree, wantDegree)
	}
} */

/* func TestAreAligned(t *testing.T) {
	//timeInterval, _ := time.DurationOf("5m")
	want := true

	got := areAligned((model.Time)1674663979.662,(model.Time)1674663979.162)

	//fail on purpose because $1 should not be replaced
	if strings.Contains(got, want) {
		t.Errorf("SubstitueVars() = %q, want %q", got, want)
	}
} */
