package utils

import (
	"resource-model-exporter/pkg/types"
	"os"
	"io/ioutil"
	"gopkg.in/yaml.v2"

	log "github.com/sirupsen/logrus"
)

//helpers
func ReadFile(filename string) []byte {
	file, err := os.Open(filename)
	if err != nil {
		log.Error(err)
		return nil
	} else {
		log.Info("Successfully Opened " + filename)
	}

	defer file.Close()
	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		log.Error(err)
		return nil
	}
	return byteValue
}

func OpenPredictors(filename string) []types.PredictorVarQueries {
	byteValue := ReadFile(filename)
	var result []types.PredictorVarQueries
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

func OpenObserved(filename string) []types.ObservedVarQueries {
	byteValue := ReadFile(filename)
	var result []types.ObservedVarQueries
	yaml.Unmarshal(byteValue, &result)
	if len(result) == 0 {
		log.Error(filename, " is empty or not well formated")
	}
	for i := 0; i < len(result); i++ {
		log.Debug("Resource Name: " + result[i].Name)
		log.Debug("Resource Unit: " + result[i].Unit)
		log.Debug("Resource Query: " + result[i].Query)
	}
	return result
}

func OpenControl(filename string) []types.ControlVarQueries {
	byteValue := ReadFile(filename)
	var result []types.ControlVarQueries
	yaml.Unmarshal(byteValue, &result)
	if len(result) == 0 {
		log.Error(filename, " is empty or not well formated")
	}
	for i := 0; i < len(result); i++ {
		log.Debug("Control Name: " + result[i].Name)
		log.Debug("Control Query: " + result[i].Query)
	}
	return result
}