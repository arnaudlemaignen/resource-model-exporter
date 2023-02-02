package types

import "github.com/prometheus/common/model"

type PredictorMeasurement struct {
	Predictor         string
	Values            model.Value
	PolynomialDegrees []int
}

type Measurement struct {
	Predictors []PredictorMeasurement
	Usage      model.Value
}

type MeasurementConf struct {
	ImageVersion string
	CpuModel     string
	NodeType     string
}
