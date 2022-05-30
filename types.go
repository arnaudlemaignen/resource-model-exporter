package main

import (
	"time"

	"github.com/prometheus/common/model"
)

//YAML Config
type PredictorVarQueries struct {
	Vars      []Var      `yaml:"vars"`
	Resources []Resource `yaml:"resources"`
}

type Resource struct {
	Name      string      `yaml:"name"`
	Predictor []Predictor `yaml:"predictors"`
}

type Predictor struct {
	Name  string `yaml:"name"`
	Query string `yaml:"query"`
}

type Var struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

//measured.yaml
type ObservedVarQueries struct {
	Name  string `yaml:"name"`
	Unit  string `yaml:"unit"`
	Query string `yaml:"query"`
}

//control.yaml
type ControlVarQueries struct {
	Name  string `yaml:"name"`
	Query string `yaml:"query"`
}

//JSON Export
type RegressionExport struct {
	Round         int
	Container     string
	Resource      string
	Image_Version string
	Timings       []Timing
	Nodes         []Node
	Namespaces    []string
	Pods          []string
	Regression    Reg
}

type Timing struct {
	Steps    string
	Duration time.Time
}

type Node struct {
	Name    string
	Type    string
	FreqGhz float64
}

type Reg struct {
	Formula            string
	Unit               string
	Predictors         []PredictorExport
	Offset             float64
	Limit              float64
	R2                 float64
	VarianceObserved   float64
	VariancePredictors float64
	N                  int
}

type PredictorExport struct {
	Name  string
	Coeff float64
}

//Measurement
type PredictorMeasurement struct {
	Predictor string
	Values    model.Value
}

type Measurement struct {
	Predictors   []PredictorMeasurement
	Usage        model.Value
	Limit        model.Value
	ImageVersion model.Value
}
