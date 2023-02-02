package types

import (
	"resource-model-exporter/pkg/utils"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type PredictorVarQueries struct {
	Vars      []Var      `yaml:"vars"`
	Resources []Resource `yaml:"resources"`
}

type Resource struct {
	Name      string      `yaml:"name"`
	Predictor []Predictor `yaml:"predictors"`
}

type Predictor struct {
	Name              string `yaml:"name"`
	Query             string `yaml:"query"`
	PolynomialDegrees []int  `yaml:"polynomial_degrees"`
}

type Var struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

func OpenPredictors(filename string) []PredictorVarQueries {
	byteValue := utils.ReadFile(filename)
	var result []PredictorVarQueries
	yaml.Unmarshal(byteValue, &result)
	if len(result) == 0 {
		log.Error(filename, " is empty or not well formated")
	}
	for i := 0; i < len(result); i++ {
		log.Debug("Var: " + result[i].Vars[0].Name)
		log.Debug("Var: " + result[i].Vars[0].Value)
		log.Debug("Resource Name: " + result[i].Resources[0].Name)
		log.Debug("Resource Predictor Name: " + result[i].Resources[0].Predictor[0].Name)
		log.Debug("Resource Predictor Query: " + result[i].Resources[0].Predictor[0].Query)
	}
	return result
}
