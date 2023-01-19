package utils

import (
	"context"
	// "os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
)

//https://stackoverflow.com/questions/16673766/basic-http-auth-in-go
//see https://godoc.org/github.com/prometheus/client_golang/api/prometheus/v1#example-API--Query
//https://github.com/prometheus/client_golang/blob/master/api/prometheus/v1/example_test.go
func Prom_query(endpoint string, query string) (model.Value, error) {
	client, err := api.NewClient(api.Config{
		Address: endpoint,
	})
	if err != nil {
		log.Error("Error creating client:", err)
		// os.Exit(1)
		return nil, err
	}

	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, warnings, err := v1api.Query(ctx, query, time.Now())
	if err != nil {
		log.Error("Error querying Prometheus:", err)
		// os.Exit(1)
		return nil, err
	}
	if len(warnings) > 0 {
		log.Warn("Warnings:", warnings)
	}
	log.Debug("Result:", result)
	return result, nil
}

func Prom_queryRange(endpoint string, query string, start time.Time, end time.Time, step time.Duration) (model.Value, error) {
	client, err := api.NewClient(api.Config{
		Address: endpoint,
	})
	if err != nil {
		log.Error("Error creating client:", err)
		// os.Exit(1)
		return nil, err
	}

	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r := v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	}
	result, warnings, err := v1api.QueryRange(ctx, query, r)
	if err != nil {
		log.Error("Error querying Prometheus:", err)
		// os.Exit(1)
		return nil, err
	}
	if len(warnings) > 0 {
		log.Warn("Warnings:", warnings)
	}
	log.Trace("Result:", result)
	return result, nil
}

func Prom_series(endpoint string, query string, start time.Time, end time.Time) (string, error) {
	client, err := api.NewClient(api.Config{
		Address: endpoint,
	})
	if err != nil {
		log.Error("Error creating client:", err)
		// os.Exit(1)
		return "nil", err
	}

	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	lbls, warnings, err := v1api.Series(ctx, []string{
		query,
	}, start, end)
	if err != nil {
		log.Error("Error querying Prometheus:", err)
		// os.Exit(1)
		return "nil", err
	}
	if len(warnings) > 0 {
		log.Warn("Warnings:", warnings)
	}
	var sb strings.Builder
	for _, lbl := range lbls {
		sb.WriteString(lbl.String())
		sb.WriteString(" ")
	}
	result := sb.String()
	log.Debug("Result:", result)
	return result, nil
}
