package collector

import (
	"math"
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
	"gonum.org/v1/gonum/stat"
)

var (
	maxSupportedDegrees = 4
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
		conf := e.MeasureResourceConfig(ch, container, pred.Vars)

		validMeasurements := 0
		//For any resources in the predictors yaml (CPU/Mem,Storage, IOPS...)
		for _, resPred := range pred.Resources {
			observed := FindObserved(resPred.Name, e.observed)
			limit := e.MeasureResourceLimit(ch, container, resPred.Name, pred.Vars)
			isValid, m := e.MeasureResource(container, resPred.Name, resPred.Predictor, observed.Query, limit, pred.Vars)
			//regression
			if isValid {
				N := len(m.Usage.(model.Matrix)[0].Values)
				log.Info("====> Regression for ", container, " ", resPred.Name)
				r := RunRegressionMLR(container, resPred.Name, m.Predictors, m.Usage)
				log.Info("Formula : ", strings.Replace(r.Formula, "Predicted = ", "", -1))
				//R2: A metric that tells us the proportion of the variance in the response variable of a regression model that can be explained by the predictor variables. This value ranges from 0 to 1. The higher the R2 value, the better a model fits a dataset.
				log.Info("R² : ", math.Floor(r.R2*10000)/100, " %, Var. Observed : ", r.Varianceobserved, ", Var. Predictors : ", r.VariancePredicted, ", N : ", N)
				log.Debug("Regression ", r)

				//trial gonum regression to compare
				//RunRegressionGonum(container, resPred.Name, m.Predictors, m.Usage)

				//TODO maybe to use long short-term memory (LSTM) network models
				//supposed to be better than MLR (see https://www.sciencedirect.com/science/article/pii/S092523122031167X)
				//https://github.com/owulveryck/lstm
				//https://subscription.packtpub.com/book/data/9781789340990/7/ch07lvl1sec32/building-an-lstm-in-gorgonia
				//or multivariate gradient descent
				//https://nulab.com/learn/software-development/gradient-descent-linear-regression-using-golang/
				//https://github.com/mogaleaf/go-linear-gradient
				//https://gorgonia.org/tutorials/iris/
				// or random forest
				//better than MLR (see https://iopscience.iop.org/article/10.1088/1742-6596/1631/1/012153/pdf)
				//https://github.com/fxsjy/RF.go

				//Multi layer perceptron
				//https://www.researchgate.net/publication/317418047_Utilization_based_prediction_model_for_resource_provisioning
				//https://github.com/Agurato/goceptron
				//https://madeddu.xyz/posts/neuralnetwork/
				//https://medium.com/@kidtronnix/making-a-neural-network-from-scratch-using-go-2bfd4dafce9c
				//https://github.com/cdipaolo/goml/tree/master/perceptron

				//most promissing framework
				//https://github.com/cdipaolo/goml
				//https://github.com/gorgonia/gorgonia

				//https://www.freecodecamp.org/news/learn-how-to-select-the-best-performing-linear-regression-for-univariate-models-e9d429c40581/
				//Only compare linear models for the same dataset.
				//Find a model with a high adjusted R2
				//Make sure this model has equally distributed residuals around zero
				//Make sure the errors of this model are within a small bandwidth

				/* 				var bestSoFarPred types.Container
				   				for _, containerBest := range *containersBest {
				   					if containerBest.Name == container {
				   						bestSoFar = containerBest
				   					}
				   				} */
				//oldBestSoFarReg := findBestSoFar(containersBest, container, resPred.Name)
				/* 				newBestSoFarReg := calculateNewBestSoFar(container, resPred.Name, r, N, len(m.Predictors), observed.Unit,
				types.InfoReg{Image: conf.ImageVersion, CpuModel: conf.CpuModel, NodeType: conf.NodeType, ROI: e.maxRoi.String(), LastUpdated: time.Now().Format("2006-01-02 15:04:05")}, containersBest)
				*/
				newBestSoFarReg := e.UpdateOutReg(container, resPred.Name, r, N, len(m.Predictors), limit, observed.Unit,
					types.InfoReg{Image: conf.ImageVersion, CpuModel: conf.CpuModel, NodeType: conf.NodeType, ROI: e.maxRoi.String(), LastUpdated: time.Now().Format("2006-01-02 15:04:05")})
				//create prom metrics
				e.CreatePromMetrics(ch, container, resPred.Name, observed.Unit, m, newBestSoFarReg)

				validMeasurements++
			}
		}
		log.Info("End measurement of : ", container, " with ", validMeasurements, " valid measures in ", time.Since(startContainer))
	}
}

func (e *Exporter) findContainerReg(container string) (int, types.ContainerReg) {
	for i, containerReg := range e.regs {
		if containerReg.Name == container {
			return i, containerReg
		}
	}
	//Not found we will create and return
	containerReg := types.ContainerReg{}
	e.regs = append(e.regs, containerReg)

	return len(e.regs) - 1, containerReg
}

func (e *Exporter) findResReg(indexContReg int, resource string) (int, types.ResReg) {
	for i, resReg := range e.regs[indexContReg].Reg {
		if resReg.Name == resource {
			return i, resReg
		}
	}
	//Not found we will create and return
	resReg := types.ResReg{}
	e.regs[indexContReg].Reg = append(e.regs[indexContReg].Reg, resReg)

	return len(e.regs[indexContReg].Reg) - 1, resReg
}

func (e *Exporter) UpdateOutReg(container string, resource string, r *regression.Regression, N int, preds int, limit float64, unit string, info types.InfoReg) types.ResReg {
	indexContReg, _ := e.findContainerReg(container)
	indexResReg, resReg := e.findResReg(indexContReg, resource)

	success, newResReg := e.UpdateEventuallyResReg(container, resource, r, N, preds, limit, unit, resReg)
	if success {
		e.regs[indexContReg].Name = container
		e.regs[indexContReg].Info = info
		e.regs[indexContReg].Reg[indexResReg] = newResReg
		types.WriteBestSoFarRegFromYaml(e.outReg, e.regs)
	}

	return resReg
}

func (e *Exporter) UpdateEventuallyResReg(container string, predictor string, r *regression.Regression, N int, preds int, limit float64, unit string, bestSoFar types.ResReg) (bool, types.ResReg) {
	if r.R2 > bestSoFar.R2 {
		log.Info("New best so far regression ", math.Floor(r.R2*10000)/100, " % | previous was ", math.Floor(bestSoFar.R2*10000)/100, " %")
		//update reg
		bestSoFar.Name = predictor
		bestSoFar.R2 = r.R2
		bestSoFar.Formula = strings.Replace(r.Formula, "Predicted = ", "", -1)
		bestSoFar.Unit = unit
		bestSoFar.VarianceObs = r.Varianceobserved
		bestSoFar.VariancePred = r.VariancePredicted
		bestSoFar.Limit = limit
		bestSoFar.N = N
		bestSoFar.Offset = r.GetCoeffs()[0]
		//prepare the dynamic
		bestSoFar.PredictorCoeffs = make([]types.PredictorCoeff, preds)
		for i := 0; i < len(bestSoFar.PredictorCoeffs); i++ {
			bestSoFar.PredictorCoeffs[i].Coeffs = make([]float64, maxSupportedDegrees)
		}
		mapPredIndices := make(map[string]int)

		// Model Regression Coeff metrics
		for i, coeff := range r.GetCoeffs() {
			if i > 0 {
				//GetVar() contains the alias of the pred with the cross info => image_pixels_M | (image_pixels_M)^2 | (image_pixels_M)^3
				predAlias, degree := getPredCoeffs(r.GetVar(i - 1))

				index := len(mapPredIndices)
				_, exists := mapPredIndices[predAlias]
				if exists {
					index = mapPredIndices[predAlias]
				} else {
					mapPredIndices[predAlias] = index
				}

				bestSoFar.PredictorCoeffs[index].Name = predAlias
				bestSoFar.PredictorCoeffs[index].Coeffs[degree-1] = coeff
			}
		}
		return true, bestSoFar
	}
	return false, bestSoFar
}

func (e *Exporter) MeasureResource(container string, resource string, preds []types.Predictor, measuredQuery string, limit float64, vars []types.Var) (bool, types.Measurement) {
	//measure predictors vars (dimensioning inputs)
	measuredPredictors := MeasurePredictors(resource, e.promURL, e.maxRoi, e.interval, preds, vars)
	//measure measured vars (resource usage)
	measuredUsage := MeasureUsage(resource, e.promURL, e.maxRoi, e.interval, measuredQuery, vars)

	isValid := e.ValidateMeasurement(container, resource, measuredPredictors, measuredUsage, limit)
	var m types.Measurement
	m.Predictors = measuredPredictors
	m.Usage = measuredUsage

	return isValid, m
}

func (e *Exporter) MeasureResourceConfig(ch chan<- prometheus.Metric, container string, vars []types.Var) types.MeasurementConf {
	//guse instant query (no need for historical)
	//get the image version
	//get cpu model
	//   needs extra node_exporter --collector.cpu.info flag
	//   node_cpu_info{cachesize="3072 KB", core="0", cpu="0", family="6", instance="localhost:9100", job="node-exporter", microcode="0xffffffff", model="61", model_name="Intel(R) Core(TM) i5-5200U CPU @ 2.20GHz", package="0", stepping="4", vendor="GenuineIntel"}
	//get AWS instance Type
	//   node_boot_time_seconds{kubernetes_io_arch="amd64",node_kubernetes_io_instance_type="r5a.4xlarge"}

	var c types.MeasurementConf
	c.ImageVersion = GetLabel(FindInfoLabel("image_version", e.info), MeasureControl(e.promURL, FindInfoQuery("image_version", e.info), vars, e.interval))
	c.CpuModel = GetLabel(FindInfoLabel("cpu_model", e.info), MeasureControl(e.promURL, FindInfoQuery("cpu_model", e.info), vars, e.interval))
	c.NodeType = GetLabel(FindInfoLabel("node_type", e.info), MeasureControl(e.promURL, FindInfoQuery("node_type", e.info), vars, e.interval))

	ch <- prometheus.MustNewConstMetric(
		metricMeasurementConfig, prometheus.GaugeValue, 1, container, c.ImageVersion, c.NodeType, c.CpuModel,
	)

	return c
}

func (e *Exporter) MeasureResourceLimit(ch chan<- prometheus.Metric, container string, res string, vars []types.Var) float64 {
	//get the CPU/Mem limits to check we are under... use instant query (no need for historical)
	limit := -1.0

	query := FindLimitQuery(res, e.limits)
	if query != "" {
		limit = GetValue(MeasureControl(e.promURL, query, vars, e.interval))
		unit := FindLimitUnit(res, e.limits)

		ch <- prometheus.MustNewConstMetric(
			metricLimit, prometheus.GaugeValue, limit, container, res, unit,
		)
	}

	return limit
}

func (e *Exporter) CreatePromMetrics(ch chan<- prometheus.Metric, container string, resource string, unit string, measurement types.Measurement, reg types.ResReg) {
	measuredPredictors := measurement.Predictors
	measuredObserved := measurement.Usage
	//last value of the observed or Pred (same size)
	lastValueIndex := len(measuredObserved.(model.Matrix)[0].Values) - 1

	//Offset
	ch <- prometheus.MustNewConstMetric(
		metricModelCoeffs, prometheus.GaugeValue, reg.Offset, container, resource, "offset", "offset", "0",
	)

	// Model Regression Coeff metrics
	for i, pred := range reg.PredictorCoeffs {
		//degree is starting at 1
		for degree, coeff := range pred.Coeffs {
			ch <- prometheus.MustNewConstMetric(
				metricModelCoeffs, prometheus.GaugeValue, coeff, container, resource, "predictor_"+strconv.Itoa(i), pred.Name, strconv.Itoa(degree+1),
			)
		}
	}

	ch <- prometheus.MustNewConstMetric(
		metricModelR2, prometheus.GaugeValue, reg.R2, container, resource, reg.Formula,
	)

	ch <- prometheus.MustNewConstMetric(
		metricModelVariance, prometheus.GaugeValue, reg.VarianceObs, container, resource, "observed",
	)

	ch <- prometheus.MustNewConstMetric(
		metricModelVariance, prometheus.GaugeValue, reg.VariancePred, container, resource, "predicted",
	)

	//Add number of values
	ch <- prometheus.MustNewConstMetric(
		metricModelN, prometheus.GaugeValue, float64(reg.N), container, resource,
	)

	//Add last usage value
	ch <- prometheus.MustNewConstMetric(
		metricObserved, prometheus.GaugeValue, float64(measuredObserved.(model.Matrix)[0].Values[lastValueIndex].Value), container, resource, unit,
	)

	//Add prediction based on the last predictor values
	floatLastPredictorValues := make([]float64, len(measuredPredictors))
	for j, pred := range measuredPredictors {
		floatLastPredictorValues[j] = float64(pred.Values.(model.Matrix)[0].Values[lastValueIndex].Value)
		//Add Predictors values for analysis
		ch <- prometheus.MustNewConstMetric(
			metricPredictor, prometheus.GaugeValue, floatLastPredictorValues[j], container, resource, "predictor_"+strconv.Itoa(j), pred.Predictor,
		)
	}

	//Calculate Prediction
	prediction, err := Predict(reg, floatLastPredictorValues)
	if err == nil {
		ch <- prometheus.MustNewConstMetric(
			metricPrediction, prometheus.GaugeValue, prediction, container, resource, unit, reg.Formula,
		)
	} else {
		log.Warn("Prediction was not successful for container ", container, " resource ", resource, " err ", err)
	}
}

func Predict(reg types.ResReg, floatLastPredictorValues []float64) (float64, error) {
	sum := float64(reg.Offset)
	for i, pred := range reg.PredictorCoeffs {
		//degree is starting at 1
		for degree, coeff := range pred.Coeffs {
			sum += coeff * math.Pow(floatLastPredictorValues[i], float64(degree+1))
		}
	}

	return sum, nil
}

func getPredCoeffs(coeffName string) (string, int) {
	//Find the right coeff and degree
	//GetVar() contains the alias of the pred with the cross info => image_pixels_M | (image_pixels_M)^2 | (image_pixels_M)^3
	predAlias := "NA"
	degree := 0
	if strings.Contains(coeffName, "(") && strings.Contains(coeffName, ")^") {
		coeffNameClean := coeffName[strings.LastIndex(coeffName, "(")+1 : strings.LastIndex(coeffName, ")^")]
		predAlias = coeffNameClean
		degree, _ = strconv.Atoi(coeffName[strings.LastIndex(coeffName, ")^")+2:])
	} else {
		predAlias = coeffName
		degree = 1
	}

	return predAlias, degree
}

func predictorsToMap(measuredPredictors []types.PredictorMeasurement) map[string]string {
	result := make(map[string]string)

	for j, pred := range measuredPredictors {
		result[pred.Predictor] = "predictor_" + strconv.Itoa(j)
	}

	return result
}

//MEASUREMENTS
func (e *Exporter) ValidateMeasurement(container string, resource string, measuredPredictors []types.PredictorMeasurement, measuredUsage model.Value, limit float64) bool {
	//validate that there is the same number of measurements for predictors and observed
	//validate measurement based on limits
	//if no kubestate => assume no limit, resources are not bounded
	maxValidResourceUsageThreshold := 0.9 //90%
	//validate the size of measures

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

	//to check that the data which is measured is well in the expected range
	first := measuredUsage.(model.Matrix)[0].Values[0].Timestamp
	last := measuredUsage.(model.Matrix)[0].Values[len(measuredUsage.(model.Matrix)[0].Values)-1].Timestamp
	log.Info("Measurement ROI : to measure ", e.maxRoi, ", measured ", model.Time.Sub(last, first), ", Now - measured ", model.Now().Sub(first))

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

	if len(newMeasuredUsage) == len(newMeasuredPredictor) && len(newMeasuredUsage) > 0 {
		log.Debug("Alignment Success : Observed Size is ", len(newMeasuredUsage), ", aligned in ", time.Since(start))
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

func findMax(promValues model.Value) float64 {
	max := 0.0
	for i, e := range promValues.(model.Matrix)[0].Values {
		if i == 0 || float64(e.Value) > max {
			max = float64(e.Value)
		}
	}
	return max
}

//Mulitple Linear Regression (MLR) using Least Squares
//see github.com/sajari/regression doc
func RunRegressionMLR(container string, resource string, measuredPredictors []types.PredictorMeasurement, measuredUsage model.Value) *regression.Regression {
	r := new(regression.Regression)
	r.SetObserved("Resource model for container " + container + " resource " + resource + "\n")
	//Setup predictor names
	for i := 0; i < len(measuredPredictors); i++ {
		r.SetVar(i, measuredPredictors[i].Predictor)

		// Feature cross based on computing the power of an input.
		// It also exists MultiplierCross(0, 1, 3)
		// Feature cross based on the multiplication of multiple inputs.
		for _, degree := range measuredPredictors[i].PolynomialDegrees {
			//by default degree 0 and 1 included
			if degree > 1 {
				r.AddCross(regression.PowCross(i, float64(degree)))
			}
		}
	}

	//floatData is the raw values with both observed[0] and variables[1..n]
	floatData := make([][]float64, len(measuredUsage.(model.Matrix)[0].Values))
	for i, obs := range measuredUsage.(model.Matrix)[0].Values {
		//floatDataLine is the raw Line values with both observed[0] and variables[1..n]
		floatDataLine := make([]float64, len(measuredPredictors)+1)
		//populate Observed
		floatDataLine[0] = float64(obs.Value)
		for j, pred := range measuredPredictors {
			predValues := pred.Values.(model.Matrix)[0].Values

			floatDataLine[j+1] = float64(predValues[i].Value)
		}
		floatData[i] = floatDataLine
	}
	//observed is the first cols
	r.Train(regression.MakeDataPoints(floatData, 0)...)
	r.Run()

	return r
}

//regression based on Gonum
func RunRegressionGonum(container string, resource string, measuredPredictors []types.PredictorMeasurement, measuredUsage model.Value) {
	log.Info("Regression GONUM")
	origin := false
	var weights []float64
	xs := make([]float64, len(measuredUsage.(model.Matrix)[0].Values))
	ys := make([]float64, len(measuredUsage.(model.Matrix)[0].Values))

	for i, obs := range measuredUsage.(model.Matrix)[0].Values {
		ys[i] = float64(obs.Value)
		xs[i] = float64(measuredPredictors[0].Values.(model.Matrix)[0].Values[i].Value)
	}

	alpha, beta := stat.LinearRegression(xs, ys, weights, origin)
	r2 := stat.RSquared(xs, ys, weights, alpha, beta)
	log.Info("Formula : Predicted = ", alpha, " + image_pixels_M*", beta)
	log.Info("R² : ", math.Floor(r2*10000)/100, " %")
}

//measure helpers
func MeasurePredictors(resource string, promURL string, maxRoi time.Duration, interval time.Duration, preds []types.Predictor, vars []types.Var) []types.PredictorMeasurement {
	log.Debug("=> Measure ", resource, " predictors")
	var result []types.PredictorMeasurement
	for _, pred := range preds {
		log.Debug("==> Measure ", resource, " predictor ", pred.Name)
		query := SubstitueVars(pred.Query, vars, interval)
		data := QueryRange(promURL, query, maxRoi, interval)
		if data != nil {
			result = append(result, types.PredictorMeasurement{Predictor: pred.Name, PolynomialDegrees: pred.PolynomialDegrees, Values: data})
		}
	}

	return result
}

func MeasureUsage(resource string, promURL string, maxRoi time.Duration, interval time.Duration, query string, vars []types.Var) model.Value {
	log.Debug("===> Measure ", resource, " usage")
	query = SubstitueVars(query, vars, interval)
	data := QueryRange(promURL, query, maxRoi, interval)
	return data
}

func MeasureControl(promURL string, query string, vars []types.Var, interval time.Duration) model.Value {
	log.Debug("===> Measure control")
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
		log.Debug("Sucessful query range : ", query)
	}
	return data
}

func QueryInstant(promURL string, query string) model.Value {
	//TODO change start and step with min/max roi and aggr
	data, err := utils.Prom_query(promURL, query)
	if err != nil {
		log.Error("PromQL query wrong for ", query)
	} else {
		log.Debug("Sucessful query : ", query)
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
	return ""
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

func FindLimitQuery(name string, limitVars []types.LimitVarQueries) string {
	for _, limit := range limitVars {
		if limit.Name == name {
			return limit.Query
		}
	}

	//on purpose limit is optional
	log.Debug("Could not find the query in limit.json for ", name)
	return ""
}

func FindLimitUnit(name string, limitVars []types.LimitVarQueries) string {
	for _, limit := range limitVars {
		if limit.Name == name {
			return limit.Unit
		}
	}

	log.Error("Could not find the unit in limit.json for ", name)
	return ""
}

func FindInfoQuery(name string, infoVars []types.InfoVarQueries) string {
	for _, info := range infoVars {
		if info.Name == name {
			return info.Query
		}
	}

	log.Error("Could not find the query in info.json for ", name)
	return ""
}

func FindInfoLabel(name string, infoVars []types.InfoVarQueries) string {
	for _, info := range infoVars {
		if info.Name == name {
			return info.Label
		}
	}

	log.Error("Could not find the label in info.json for ", name)
	return ""
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
