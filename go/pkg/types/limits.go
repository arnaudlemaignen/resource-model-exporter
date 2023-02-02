package types

import (
	"resource-model-exporter/pkg/utils"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type LimitVarQueries struct {
	Name  string `yaml:"name"`
	Unit  string `yaml:"unit"`
	Query string `yaml:"query"`
}

func OpenLimits(filename string) []LimitVarQueries {
	byteValue := utils.ReadFile(filename)
	var result []LimitVarQueries
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
