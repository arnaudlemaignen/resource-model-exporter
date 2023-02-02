package types

import (
	"resource-model-exporter/pkg/utils"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type InfoVarQueries struct {
	Name  string `yaml:"name"`
	Label string `yaml:"label"`
	Query string `yaml:"query"`
}

func OpenInfo(filename string) []InfoVarQueries {
	byteValue := utils.ReadFile(filename)
	var result []InfoVarQueries
	yaml.Unmarshal(byteValue, &result)
	if len(result) == 0 {
		log.Error(filename, " is empty or not well formated")
	}
	for i := 0; i < len(result); i++ {
		log.Debug("Info Name: " + result[i].Name)
		log.Debug("Info Query: " + result[i].Query)
	}
	return result
}
