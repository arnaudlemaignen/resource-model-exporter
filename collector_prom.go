package main

import (
	"encoding/json"
	"io/ioutil"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/sajari/regression"
	log "github.com/sirupsen/logrus"
)

func (e *Exporter) CollectPromMetrics(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		metricConfig, prometheus.GaugeValue, 1, e.version, e.minRoi.String(), e.maxRoi.String(), e.interval.String(),
	)

	e.HitProm(ch)
}

// most important godoc
// https://godoc.org/github.com/prometheus/common/model#Vector
// https://godoc.org/github.com/prometheus/client_golang/api/prometheus/v1

//Maybe to consider CPU throttling ?
// container_cpu_cfs_throttled_seconds_total
// https://itnext.io/k8s-monitor-pod-cpu-and-memory-usage-with-prometheus-28eec6d84729
func (e *Exporter) HitProm(ch chan<- prometheus.Metric) {
	for _, pred := range e.predictors {
		startContainer := time.Now()
		container := FindContainerName(pred)
		// export := RegressionExport{}
		var infoMeasurement model.Value
		log.Info("Begin measurement of : ", container)
		validMeasurements := 0
		//Resources today are CPU/Mem, more to come : Storage, IOPS, latency ...
		for _, resPred := range pred.Resources {
			//TODO find where we spend time
			// var timings []Timing
			observed := FindObserved(resPred.Name, e.observed)
			limitQuery := FindControlQuery(resPred.Name+"_limit", e.control)
			imageQuery := FindControlQuery("image_version", e.control)
			isValid, m := e.MeasureResource(container, resPred.Name, resPred.Predictor, observed.Query, limitQuery, imageQuery, pred.Vars)
			//regression
			if isValid {
				r := RunRegression(container, resPred.Name, m.Predictors, m.Usage)
				//TODEL
				log.Info("Regression formula for ", container, " ", resPred.Name, " : ", r.Formula)
				log.Info("Regression stats for ", container, " ", resPred.Name, ", R² : ", r.R2, ", Var. Observed : ", r.Varianceobserved, ", Var. Predictors : ", r.VariancePredicted)
				log.Debug("Regression ", r)
				//create prom metrics
				e.CreatePromMetrics(ch, container, resPred.Name, observed.Unit, m.Predictors, r)
				infoMeasurement = m.ImageVersion
				//export data
				// TODO Namespaces,Pods,Nodes,
				limit := 0.0
				if len(m.Limit.(model.Vector)) > 0 {
					limit = float64(m.Limit.(model.Vector)[0].Value)
				}
				ExportResultToJson(RegressionExport{Round: round, Container: container, Resource: resPred.Name, Image_Version: "TODO",
					Regression: Reg{Formula: r.Formula, Unit: observed.Unit, Limit: limit,
						R2: r.R2, VarianceObserved: r.Varianceobserved, VariancePredictors: r.VariancePredicted}})
				validMeasurements++
			}
		}

		if validMeasurements > 0 {
			e.CreatePromConfigMetric(ch, container, infoMeasurement)
		}

		log.Info("End measurement of : ", container, " with ", validMeasurements, " valid measures in ", time.Since(startContainer))
	}
}

func (e *Exporter) MeasureResource(container string, resource string, preds []Predictor, measuredQuery string, limitQuery string, imageQuery string, vars []Var) (bool, Measurement) {
	//measure predictors vars (dimensioning inputs)
	measuredPredictors := MeasurePredictors(resource, e.promURL, e.maxRoi, e.interval, preds, vars)
	//measure measured vars (resource usage)
	measuredUsage := MeasureUsage(resource, e.promURL, e.maxRoi, e.interval, measuredQuery, vars)

	//get the limits to check we are under... use instant query (no need for historical)
	measuredLimit := MeasureControl(resource, e.promURL, limitQuery, vars)
	//get the image version
	measuredImageVersion := MeasureControl(resource, e.promURL, imageQuery, vars)
	//TODO
	// get HW information
	// get AWS instance Type
	// get CPU Ghz

	isValid := ValidateMeasurement(container, resource, measuredPredictors, measuredUsage, measuredLimit)
	var m Measurement
	m.Predictors = measuredPredictors
	m.Usage = measuredUsage
	m.Limit = measuredLimit
	m.ImageVersion = measuredImageVersion

	return isValid, m
}

func (e *Exporter) CreatePromMetrics(ch chan<- prometheus.Metric, container string, resource string, unit string, measuredPredictors []PredictorMeasurement, r *regression.Regression) {
	// Model Regression Coeff metrics
	for i, coeff := range r.GetCoeffs() {
		if i == 0 {
			ch <- prometheus.MustNewConstMetric(
				metricModelCoeffs, prometheus.GaugeValue, coeff, container, resource, "offset",
			)
		} else {
			ch <- prometheus.MustNewConstMetric(
				metricModelCoeffs, prometheus.GaugeValue, coeff, container, resource, r.GetVar(i-1),
			)
		}
	}

	ch <- prometheus.MustNewConstMetric(
		metricModelR2, prometheus.GaugeValue, r.R2, container, resource, r.Formula,
	)

	ch <- prometheus.MustNewConstMetric(
		metricModelVariance, prometheus.GaugeValue, r.Varianceobserved, container, resource, "observed",
	)

	ch <- prometheus.MustNewConstMetric(
		metricModelVariance, prometheus.GaugeValue, r.VariancePredicted, container, resource, "predicted",
	)

	//Add number of values
	ch <- prometheus.MustNewConstMetric(
		metricModelN, prometheus.GaugeValue, float64(len(measuredPredictors[0].Values.(model.Matrix)[0].Values)), container, resource,
	)

	//Add prediction based on the last predictor values
	floatLastPredictorValues := make([]float64, len(measuredPredictors))
	for j, pred := range measuredPredictors {
		floatLastPredictorValues[j] = float64(pred.Values.(model.Matrix)[0].Values[0].Value)
	}
	prediction, err := r.Predict(floatLastPredictorValues)
	if err != nil {
		ch <- prometheus.MustNewConstMetric(
			metricPrediction, prometheus.GaugeValue, prediction, container, resource, unit, r.Formula,
		)
	} else {
		log.Warn("Prediction was not successful for container ", container, " resource ", resource)
	}

}

func (e *Exporter) CreatePromConfigMetric(ch chan<- prometheus.Metric, container string, infoMeasurement model.Value) {
	//TODO finish the labels
	image_version := "unknown"

	if len(infoMeasurement.(model.Vector)) > 0 {
		vectorVal := infoMeasurement.(model.Vector)
		for _, elem := range vectorVal {
			image_version = string(elem.Metric["image_version"])
		}

		ch <- prometheus.MustNewConstMetric(
			metricMeasurementConfig, prometheus.GaugeValue, 1, "namespace", "pod", container, image_version, "node", "node_type", "cpu_freq_ghz",
		)
	}
}

//MEASUREMENTS
func ValidateMeasurement(container string, resource string, measuredPredictors []PredictorMeasurement, measuredUsage model.Value, measuredLimit model.Value) bool {
	//validate that there is the same number of measurements for predictors and observed
	//validate measurement based on limits
	//if no kubestate => assume no limit, resources are not bounded
	maxValidResourceUsageThreshold := 0.9 //90%
	//validate the size of measures
	// log.Info("measuredUsage ", measuredUsage)

	//check that there is at least 1 and only 1 serie returning
	if len(measuredUsage.(model.Matrix)) != 1 {
		log.Info("The query was not returning 1 serie, ", len(measuredUsage.(model.Matrix)), " instead")
		return false
	}

	observedSize := len(measuredUsage.(model.Matrix)[0].Values)
	for _, pred := range measuredPredictors {
		if len(pred.Values.(model.Matrix)) == 0 {
			log.Error("Observed Size was ", observedSize, " but Predictor size was 0 for PredictorName ", pred.Predictor, " for container ", container, " resource ", resource)
			return false
		}
		predictorSize := len(pred.Values.(model.Matrix)[0].Values)
		if predictorSize != observedSize {
			log.Warn("Observed Size was ", observedSize, " but Predictor size was ", predictorSize, " for PredictorName ", pred.Predictor, " for container ", container, " resource ", resource)
			// //We will limit the number of entries to the smallest slice
			// delta := observedSize - predictorSize
			// // pred = (*pred)[delta:]
			// newPredSlice := pred.Values
			// newPredSlice = append(newPredSlice[1:delta], newPredSlice[delta:]...)
			return false
		}
	}

	//validate that the observed measures where not exceeding 90% of limit
	if len(measuredLimit.(model.Vector)) > 0 {
		limit := float64(measuredLimit.(model.Vector)[0].Value)
		//find max resource MeasureUsage
		max := FindMax(measuredUsage)
		if max > 0.9*limit {
			log.Warn("Observed Limit was ", limit, " but Max Observed Usage ", max, " which is > to the threshold of ", maxValidResourceUsageThreshold, " for container ", container, " resource ", resource)
			return false
		}
	} else {
		log.Info("No limit found for container ", container, " resource ", resource)
	}

	return true
}

func FindMax(promValues model.Value) float64 {
	max := 0.0
	for i, e := range promValues.(model.Matrix)[0].Values {
		if i == 0 || float64(e.Value) > max {
			max = float64(e.Value)
		}
	}
	// for i, e := range promValues.(model.Matrix) {
	// 	if i == 0 || float64(e.Value) > max {
	// 		max = float64(e.Value)
	// 	}
	// }
	return max
}

//regression helper
//see github.com/sajari/regression doc
//see github.com/haarts/regression
func RunRegression(container string, resource string, measuredPredictors []PredictorMeasurement, measuredUsage model.Value) *regression.Regression {
	r := new(regression.Regression)
	r.SetObserved("Resource model for container " + container + " resource " + resource + "\n")
	//Setup predictor names
	// var dataPoints []regression.DataPoint
	for i := 0; i < len(measuredPredictors); i++ {
		r.SetVar(i, measuredPredictors[i].Predictor)
	}

	//floatData is the raw values with both observed[0] and variables[1..n]
	floatData := make([][]float64, len(measuredUsage.(model.Matrix)[0].Values))
	log.Info("Regression for resource: ", resource, ", container: ", container, ", N: ", len(floatData))
	for i, obs := range measuredUsage.(model.Matrix)[0].Values {
		//floatDataLine is the raw Line values with both observed[0] and variables[1..n]
		floatDataLine := make([]float64, len(measuredPredictors)+1)
		//populate Observed
		floatDataLine[0] = float64(obs.Value)
		for j, pred := range measuredPredictors {
			predValues := pred.Values.(model.Matrix)[0].Values

			floatDataLine[j+1] = float64(predValues[i].Value)
		}
		// floatDataLine[i] = float64(obs.Value)
		// for j, pred := range measuredPredictors {
		// 	floatDataLine[j+1] = float64(pred.Values.(model.Vector)[i].Value)
		// }
		// floatData = append(floatData, floatDataLine)
		floatData[i] = floatDataLine
		// log.Info("len floatData ", len(floatData))
	}

	//observed is the first cols
	r.Train(regression.MakeDataPoints(floatData, 0)...)
	r.Run()

	return r
}

//TO REMOVE
//see github.com/haarts/regression
// 	//Setup values
// 	var d *regression.dataPoint
//
// 	// d.Variables = make([]float64, len(measuredPredictors)-1)
// 	// for i := 0; i < len(measuredPredictors); i++ {
// 	// 	vectorVal := measuredPredictors[i].Values
// 	// 	floatData := make([]float64, len(vectorVal))
// 	// 	for j := 0; j < len(vectorVal); j++ {
// 	// 		floatData[j] = float64(vectorVal[j].Value)
// 	// 	}
// 	// 	d.Variables[i] = floatData
// 	// }
// 	//
// 	// floatData := make([]float64, len(measuredUsage))
// 	// for j := 0; j < len(measuredUsage); j++ {
// 	// 	floatData[j] = float64(measuredUsage[j].Value)
// 	// }
// 	// d.Observed = floatData
//
// 	//Train
// 	r.Train(d)
// 	r.Run()
//
// 	return r
// }

//measure helpers
func MeasurePredictors(resource string, promURL string, maxRoi time.Duration, interval time.Duration, preds []Predictor, vars []Var) []PredictorMeasurement {
	log.Info("=> Measure ", resource, " predictors")
	var result []PredictorMeasurement
	for _, pred := range preds {
		log.Info("==> Measure ", resource, " predictor ", pred.Name)
		query := SubstitueVars(pred.Query, vars)
		data := QueryRange(promURL, query, maxRoi, interval)
		if data != nil {
			result = append(result, PredictorMeasurement{pred.Name, data})
		}
	}

	return result
}

func MeasureUsage(resource string, promURL string, maxRoi time.Duration, interval time.Duration, query string, vars []Var) model.Value {
	log.Info("===> Measure ", resource, " usage")
	query = SubstitueVars(query, vars)
	data := QueryRange(promURL, query, maxRoi, interval)
	return data
}

func MeasureControl(control string, promURL string, query string, vars []Var) model.Value {
	log.Info("===> Measure ", control, " control")
	query = SubstitueVars(query, vars)
	data := QueryInstant(promURL, query)
	return data
}

//misc helpers
func QueryRange(promURL string, query string, start time.Duration, step time.Duration) model.Value {
	data, err := Prom_queryRange(promURL, query, time.Now().Add(-start), time.Now(), step)
	if err != nil {
		log.Error("PromQL query range wrong for ", query)
	} else {
		log.Info("Sucessful query range : ", query)
	}
	return data
}

func QueryInstant(promURL string, query string) model.Value {
	//TODO change start and step with min/max roi and aggr
	data, err := Prom_query(promURL, query)
	if err != nil {
		log.Error("PromQL query wrong for ", query)
	} else {
		log.Info("Sucessful query : ", query)
	}
	return data
}

func SubstitueVars(query string, vars []Var) string {
	//add Global Vars to Vars
	vars = append(vars, Var{Name: "interval", Value: interval})
	for _, variable := range vars {
		query = strings.Replace(query, "$"+variable.Name, variable.Value, -1)
	}
	//no $ should be in query now
	if strings.Contains(query, "$") {
		log.Error("Missing a var for query ", query)
	}
	return query
}

func FindContainerName(pred PredictorVarQueries) string {
	for _, variable := range pred.Vars {
		if variable.Name == "container" {
			return variable.Value
		}
	}

	log.Error("Could not find the container name in predictors.json for predictor ", pred)
	return "nil"
}

func FindObserved(resPred string, observedVars []ObservedVarQueries) ObservedVarQueries {
	for _, observed := range observedVars {
		if observed.Name == resPred {
			return observed
		}
	}

	log.Error("Could not find the query in measured.json for resource ", resPred)
	return ObservedVarQueries{Query: "N/A", Unit: "N/A"}
}

func FindControlQuery(name string, controlVars []ControlVarQueries) string {
	for _, control := range controlVars {
		if control.Name == name {
			return control.Query
		}
	}

	log.Error("Could not find the query in control.json for control ", name)
	return "nil"
}

func ExportResultToJson(reg RegressionExport) {
	file, _ := json.MarshalIndent(reg, "", " ")
	_ = ioutil.WriteFile("test.json", file, 0644)
}
