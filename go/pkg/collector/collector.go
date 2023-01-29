package collector

import (
	"resource-model-exporter/pkg/types"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (
	round     = 0
	namespace = "res_model"
	//Input Metrics
	metricConfig = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "config"),
		"Main configuration details",
		[]string{"version", "max_roi", "interval"}, nil,
	)
	metricMeasurementConfig = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "measurement_config"),
		"Measurement configuration details",
		[]string{"container", "image_version", "node_type", "cpu_model"}, nil,
	)

	metricPredictor = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "predictor_usage"),
		"Predictor resource measured usage for the last ROI",
		[]string{"container", "resource", "coeff_name", "coeff_alias"}, nil,
	)
	metricObserved = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "observed_usage"),
		"Observed resource measured usage for the last ROI",
		[]string{"container", "resource", "unit"}, nil,
	)
	metricLimit = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "limits"),
		"Limit resource measured for the last ROI",
		[]string{"container", "resource", "unit"}, nil,
	)

	//Output Metrics
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
)

type Exporter struct {
	promURL, version string
	maxRoi, interval time.Duration
	predictors       []types.PredictorVarQueries
	observed         []types.ObservedVarQueries
	info             []types.InfoVarQueries
	limits           []types.LimitVarQueries
}

func NewExporter(promURL string, version string, maxRoi time.Duration, interval time.Duration, predictors []types.PredictorVarQueries, observed []types.ObservedVarQueries, info []types.InfoVarQueries, limits []types.LimitVarQueries) *Exporter {
	return &Exporter{
		promURL:    promURL,
		version:    version,
		maxRoi:     maxRoi,
		interval:   interval,
		predictors: predictors,
		observed:   observed,
		info:       info,
		limits:     limits,
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- metricConfig
	ch <- metricMeasurementConfig
	ch <- metricLimit
	ch <- metricObserved
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
