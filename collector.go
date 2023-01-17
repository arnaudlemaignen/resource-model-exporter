package main

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (
	namespace = "res_model"
	// Metrics
	metricConfig = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "config"),
		"Main configuration details",
		[]string{"version", "min_roi", "max_roi", "interval"}, nil,
	)
	metricMeasurementConfig = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "measurement_config"),
		"Measurement configuration details",
		[]string{"namespace", "pod", "container", "image_version", "node", "node_type", "cpu_freq_ghz"}, nil,
	)

	metricPrediction = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "prediction_usage"),
		"Prediction of resource usage based on resource model regression for the last ROI",
		[]string{"container", "resource", "unit", "formula"}, nil,
	)

	metricModelCoeffs = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "regression_coeff"),
		"Resource model regression coeff for the last ROI",
		[]string{"container", "resource", "coeff_name", "coeff_alias"}, nil,
	)

	metricModelR2 = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "regression_r2"),
		"Resource model regression determination coeff R2 for the last ROI",
		[]string{"container", "resource", "formula"}, nil,
	)
	metricModelVariance = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "regression_variance"),
		"Resource model regression variance observed/predictors for the last ROI",
		[]string{"container", "resource", "variance_name"}, nil,
	)
	metricModelN = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "regression_n"),
		"Resource model regression number of samples for the last ROI",
		[]string{"container", "resource"}, nil,
	)
	metricPredictor = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "predictor_usage"),
		"Predictor resource measured usage for the last ROI",
		[]string{"container", "resource", "coeff_name", "coeff_alias"}, nil,
	)
)

type Exporter struct {
	promURL, version         string
	minRoi, maxRoi, interval time.Duration
	predictors               []PredictorVarQueries
	observed                 []ObservedVarQueries
	control                  []ControlVarQueries
}

func NewExporter(promURL string, version string, minRoi time.Duration, maxRoi time.Duration, interval time.Duration, predictors []PredictorVarQueries, observed []ObservedVarQueries, control []ControlVarQueries) *Exporter {
	return &Exporter{
		promURL:    promURL,
		version:    version,
		minRoi:     minRoi,
		maxRoi:     maxRoi,
		interval:   interval,
		predictors: predictors,
		observed:   observed,
		control:    control,
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- metricConfig
	ch <- metricMeasurementConfig
	ch <- metricPrediction
	ch <- metricModelCoeffs
	ch <- metricModelR2
	ch <- metricModelVariance
	ch <- metricModelN

}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	startProm := time.Now()
	e.CollectPromMetrics(ch)
	end := time.Now()

	log.Info("Collect round ", round, " finished in ", end.Sub(startProm))
	round++
}
