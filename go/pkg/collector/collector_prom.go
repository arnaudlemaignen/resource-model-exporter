package collector

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"regexp"
	"resource-model-exporter/pkg/types"
	"resource-model-exporter/pkg/utils"
	"strconv"
	"strings"

	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/sajari/regression"
	log "github.com/sirupsen/logrus"
)

func (e *Exporter) CollectPromMetrics(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		metricConfig, prometheus.GaugeValue, 1, e.version, e.maxRoi.String(), e.interval.String(),
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
		log.Info("Begin measurement of : ", container)
		c := e.MeasureResourceConfig(pred.Vars)
		e.CreatePromConfigMetric(ch, container, c)

		validMeasurements := 0
		//Resources today are CPU/Mem, more to come : Storage, IOPS, latency ...
		for _, resPred := range pred.Resources {
			//TODO find where we spend time
			// var timings []Timing
			observed := FindObserved(resPred.Name, e.observed)

			isValid, m := e.MeasureResource(container, resPred.Name, resPred.Predictor, observed.Query, c, pred.Vars)
			//regression
			if isValid {
				r := RunRegression(container, resPred.Name, m.Predictors, m.Usage)
				//TODEL
				log.Info("Regression formula for ", container, " ", resPred.Name, " : ", r.Formula)
				log.Info("Regression stats for ", container, " ", resPred.Name, ", R² : ", r.R2, ", Var. Observed : ", r.Varianceobserved, ", Var. Predictors : ", r.VariancePredicted)
				log.Debug("Regression ", r)
				//create prom metrics
				e.CreatePromMetrics(ch, container, resPred.Name, observed.Unit, m.Predictors, r)
				//export data
				ExportResultToJson(types.RegressionExport{Round: round, Container: container, Resource: resPred.Name, ImageVersion: c.ImageVersion, CpuModel: c.CpuModel, NodeType: c.NodeType,
					Regression: types.Reg{Formula: r.Formula, Unit: observed.Unit, CpuLimit: c.CpuLimit, MemLimit: c.MemLimit,
						R2: r.R2, VarianceObserved: r.Varianceobserved, VariancePredictors: r.VariancePredicted}})
				validMeasurements++
			}
		}
		log.Info("End measurement of : ", container, " with ", validMeasurements, " valid measures in ", time.Since(startContainer))
	}
}

func (e *Exporter) MeasureResource(container string, resource string, preds []types.Predictor, measuredQuery string, conf types.MeasurementConf, vars []types.Var) (bool, types.Measurement) {
	//measure predictors vars (dimensioning inputs)
	measuredPredictors := MeasurePredictors(resource, e.promURL, e.maxRoi, e.interval, preds, vars)
	//measure measured vars (resource usage)
	measuredUsage := MeasureUsage(resource, e.promURL, e.maxRoi, e.interval, measuredQuery, vars)

	isValid := ValidateMeasurement(container, resource, measuredPredictors, measuredUsage, conf)
	var m types.Measurement
	m.Predictors = measuredPredictors
	m.Usage = measuredUsage

	return isValid, m
}

func (e *Exporter) MeasureResourceConfig(vars []types.Var) types.MeasurementConf {
	//get the CPU/Mem limits to check we are under... use instant query (no need for historical)
	//get the image version
	//get cpu model
	//   needs extra node_exporter --collector.cpu.info flag
	//   node_cpu_info{cachesize="3072 KB", core="0", cpu="0", family="6", instance="localhost:9100", job="node-exporter", microcode="0xffffffff", model="61", model_name="Intel(R) Core(TM) i5-5200U CPU @ 2.20GHz", package="0", stepping="4", vendor="GenuineIntel"}
	//get AWS instance Type
	//   node_boot_time_seconds{kubernetes_io_arch="amd64",node_kubernetes_io_instance_type="r5a.4xlarge"}

	var c types.MeasurementConf
	c.CpuLimit = GetValue(MeasureControl(e.promURL, FindControlQuery("cpu_limit", e.control), vars, e.interval))
	c.MemLimit = GetValue(MeasureControl(e.promURL, FindControlQuery("mem_limit", e.control), vars, e.interval))
	c.ImageVersion = GetLabel("image_version", MeasureControl(e.promURL, FindControlQuery("image_version", e.control), vars, e.interval))
	c.CpuModel = GetLabel("model_name", MeasureControl(e.promURL, FindControlQuery("cpu_model", e.control), vars, e.interval))
	c.NodeType = GetLabel("node_kubernetes_io_instance_type", MeasureControl(e.promURL, FindControlQuery("node_type", e.control), vars, e.interval))

	return c
}

func (e *Exporter) CreatePromMetrics(ch chan<- prometheus.Metric, container string, resource string, unit string, measuredPredictors []types.PredictorMeasurement, r *regression.Regression) {
	// Model Regression Coeff metrics
	for i, coeff := range r.GetCoeffs() {
		if i == 0 {
			ch <- prometheus.MustNewConstMetric(
				metricModelCoeffs, prometheus.GaugeValue, coeff, container, resource, "offset", "offset",
			)
		} else {
			ch <- prometheus.MustNewConstMetric(
				metricModelCoeffs, prometheus.GaugeValue, coeff, container, resource, "predictor_"+strconv.Itoa(i-1), r.GetVar(i-1),
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
		lastValueIndex := len(pred.Values.(model.Matrix)[0].Values) - 1
		floatLastPredictorValues[j] = float64(pred.Values.(model.Matrix)[0].Values[lastValueIndex].Value)
		//Add Predictors values for analysis
		ch <- prometheus.MustNewConstMetric(
			metricPredictor, prometheus.GaugeValue, floatLastPredictorValues[j], container, resource, "predictor_"+strconv.Itoa(j), pred.Predictor,
		)
	}

	//Calulate Prediction
	prediction, err := r.Predict(floatLastPredictorValues)
	if err == nil {
		ch <- prometheus.MustNewConstMetric(
			metricPrediction, prometheus.GaugeValue, prediction, container, resource, unit, r.Formula,
		)
	} else {
		log.Warn("Prediction was not successful for container ", container, " resource ", resource, " err ", err)
	}

}

func (e *Exporter) CreatePromConfigMetric(ch chan<- prometheus.Metric, container string, conf types.MeasurementConf) {
	cpuLimit := strconv.Itoa(int(conf.CpuLimit))
	memLimit := strconv.Itoa(int(conf.MemLimit))

	ch <- prometheus.MustNewConstMetric(
		metricMeasurementConfig, prometheus.GaugeValue, 1, container, conf.ImageVersion, cpuLimit, memLimit, conf.NodeType, conf.CpuModel,
	)
}

//MEASUREMENTS
func ValidateMeasurement(container string, resource string, measuredPredictors []types.PredictorMeasurement, measuredUsage model.Value, conf types.MeasurementConf) bool {
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

	//The regression MUST only happens when observedSize = predictorSize and != 0
	if len(measuredUsage.(model.Matrix)) == 0 {
		log.Error("Observed Size was 0 for container ", container, " resource ", resource)
		return false
	} else {
		observedSize := len(measuredUsage.(model.Matrix)[0].Values)
		for _, pred := range measuredPredictors {
			if len(pred.Values.(model.Matrix)) == 0 {
				log.Error("Predictor size was 0 for PredictorName ", pred.Predictor, " for container ", container, " resource ", resource)
				return false
			} else {
				predictorSize := len(pred.Values.(model.Matrix)[0].Values)
				if predictorSize != observedSize {
					log.Info("Observed Size is ", observedSize, " is different from Predictor size ", predictorSize, " for PredictorName ", pred.Predictor, " => Will try to align them")
					success, newMeasuredUsage, newMeasuredPredictor := alignObsPredMeasurements(measuredUsage.(model.Matrix)[0].Values, pred.Values.(model.Matrix)[0].Values)
					if success {
						measuredUsage.(model.Matrix)[0].Values = newMeasuredUsage
						pred.Values.(model.Matrix)[0].Values = newMeasuredPredictor
					} else {
						return false
					}
				}
			}
		}
	}

	limit := getLimit(resource, conf)
	//validate that the observed measures where not exceeding 90% of limit
	if limit > 0 {
		//find max resource MeasureUsage
		max := findMax(measuredUsage)
		if max > maxValidResourceUsageThreshold*limit {
			log.Warn("Observed Limit was ", limit, " but Max Observed Usage ", max, " which is > to the threshold of ", maxValidResourceUsageThreshold, " for container ", container, " resource ", resource)
			return false
		}
	} else {
		log.Info("No limit found for container ", container, " resource ", resource)
	}

	return true
}

func alignObsPredMeasurements(measuredUsage []model.SamplePair, measuredPredictor []model.SamplePair) (bool, []model.SamplePair, []model.SamplePair) {
	start := time.Now()

	//We will create new slices and add the aligned elts
	var newMeasuredUsage []model.SamplePair
	var newMeasuredPredictor []model.SamplePair

	//loop on the biggest slice
	if len(measuredPredictor) > len(measuredUsage) {
		for _, pred := range measuredPredictor {
			for _, meas := range measuredUsage {
				if areAligned(pred.Timestamp, meas.Timestamp) {
					newMeasuredUsage = append(newMeasuredUsage, meas)
					newMeasuredPredictor = append(newMeasuredPredictor, pred)
					break
				}
			}
		}
	} else {
		for _, meas := range measuredUsage {
			for _, pred := range measuredPredictor {
				if areAligned(pred.Timestamp, meas.Timestamp) {
					newMeasuredUsage = append(newMeasuredUsage, meas)
					newMeasuredPredictor = append(newMeasuredPredictor, pred)
					break
				}
			}
		}
	}

	log.Info("Observed and Predictors slices aligned in ", time.Since(start))

	if len(newMeasuredUsage) == len(newMeasuredPredictor) && len(newMeasuredUsage) > 0 {
		log.Info("Alignment Success : Observed Size is ", len(newMeasuredUsage))
		return true, newMeasuredUsage, newMeasuredPredictor
	} else {
		log.Warn("Alignment Failed : Observed Size is ", len(newMeasuredUsage), ", Predictor size is ", len(newMeasuredPredictor), ", the regression will be skipped")
		return false, nil, nil
	}
}

func areAligned(a, b model.Time) bool {
	durationTolerance := 1 * time.Second
	//The time between the observed and the predictor must be less than 1s otherwise it will be removed from the scope
	//@[1674577232.893] && @[1674577232.193] => OK
	//@[1674577230.893] && @[1674577232.893] => NOK
	if a.After(b) {
		return a.Sub(b) < durationTolerance
	} else {
		return b.Sub(a) < durationTolerance
	}
}

func getLimit(resource string, measuredConf types.MeasurementConf) float64 {
	if resource == "cpu" {
		return measuredConf.CpuLimit
	} else if resource == "mem" {
		return measuredConf.MemLimit
	} else {
		log.Warn("Limit for resource ", resource, " is not supported")
	}
	return -1.0
}

func findMax(promValues model.Value) float64 {
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
func RunRegression(container string, resource string, measuredPredictors []types.PredictorMeasurement, measuredUsage model.Value) *regression.Regression {
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
func MeasurePredictors(resource string, promURL string, maxRoi time.Duration, interval time.Duration, preds []types.Predictor, vars []types.Var) []types.PredictorMeasurement {
	log.Info("=> Measure ", resource, " predictors")
	var result []types.PredictorMeasurement
	for _, pred := range preds {
		log.Info("==> Measure ", resource, " predictor ", pred.Name)
		query := SubstitueVars(pred.Query, vars, interval)
		data := QueryRange(promURL, query, maxRoi, interval)
		if data != nil {
			result = append(result, types.PredictorMeasurement{Predictor: pred.Name, Values: data})
		}
	}

	return result
}

func MeasureUsage(resource string, promURL string, maxRoi time.Duration, interval time.Duration, query string, vars []types.Var) model.Value {
	log.Info("===> Measure ", resource, " usage")
	query = SubstitueVars(query, vars, interval)
	data := QueryRange(promURL, query, maxRoi, interval)
	return data
}

func MeasureControl(promURL string, query string, vars []types.Var, interval time.Duration) model.Value {
	log.Info("===> Measure control")
	query = SubstitueVars(query, vars, interval)
	data := QueryInstant(promURL, query)
	return data
}

//misc helpers
func QueryRange(promURL string, query string, start time.Duration, step time.Duration) model.Value {
	data, err := utils.Prom_queryRange(promURL, query, time.Now().Add(-start), time.Now(), step)
	if err != nil {
		log.Error("PromQL query range wrong for ", query)
	} else {
		log.Info("Sucessful query range : ", query)
	}
	return data
}

func QueryInstant(promURL string, query string) model.Value {
	//TODO change start and step with min/max roi and aggr
	data, err := utils.Prom_query(promURL, query)
	if err != nil {
		log.Error("PromQL query wrong for ", query)
	} else {
		log.Info("Sucessful query : ", query)
	}
	return data
}

func SubstitueVars(query string, vars []types.Var, interval time.Duration) string {
	//add Global Vars to Vars
	vars = append(vars, types.Var{Name: "interval", Value: interval.String()})
	for _, variable := range vars {
		query = strings.Replace(query, "$"+variable.Name, variable.Value, -1)
	}
	//no $ should be in query now (except $1... used in label_replace promQL)
	matched, _ := regexp.MatchString("\\$[a-z].*", query)
	if matched {
		log.Error("Missing a var for query ", query)
	}
	return query
}

func FindContainerName(pred types.PredictorVarQueries) string {
	for _, variable := range pred.Vars {
		if variable.Name == "container" {
			return variable.Value
		}
	}

	log.Error("Could not find the container name in predictors.json for predictor ", pred)
	return "nil"
}

func FindObserved(resPred string, observedVars []types.ObservedVarQueries) types.ObservedVarQueries {
	for _, observed := range observedVars {
		if observed.Name == resPred {
			return observed
		}
	}

	log.Error("Could not find the query in measured.json for resource ", resPred)
	return types.ObservedVarQueries{Query: "N/A", Unit: "N/A"}
}

func FindControlQuery(name string, controlVars []types.ControlVarQueries) string {
	for _, control := range controlVars {
		if control.Name == name {
			return control.Query
		}
	}

	log.Error("Could not find the query in control.json for control ", name)
	return "nil"
}

func ExportResultToJson(reg types.RegressionExport) {
	//const EXPORT_PATH = "export/"
	//TODO maybe have 1 file per Day and clean up to keep only last X days
	filename := "export.json"

	err := checkFile(filename)
	if err != nil {
		log.Error(err)
	}
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Error(err)
	}

	//build the Json struct
	data := []types.RegressionExport{}
	json.Unmarshal(file, &data)
	data = append(data, reg)

	// Preparing the data to be marshalled and written.
	dataBytes, err := json.Marshal(data)
	if err != nil {
		log.Error(err)
	}
	err = ioutil.WriteFile(filename, dataBytes, 0644)
	if err != nil {
		log.Error(err)
	}
}

func GetLabel(label string, metric model.Value) string {
	if len(metric.(model.Vector)) > 0 {
		vectorVal := metric.(model.Vector)
		for _, elem := range vectorVal {
			return string(elem.Metric[model.LabelName(label)])
		}
	}
	return ""
}

func GetValue(metric model.Value) float64 {
	if len(metric.(model.Vector)) > 0 {
		return float64(metric.(model.Vector)[0].Value)
	}
	return -1.0
}

func checkFile(filename string) error {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		_, err := os.Create(filename)
		if err != nil {
			return err
		}
	}
	return nil
}
