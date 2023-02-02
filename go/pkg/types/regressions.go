package types

import (
	"io/ioutil"
	"resource-model-exporter/pkg/utils"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type ContainerReg struct {
	Name string   `yaml:"name"`
	Info InfoReg  `yaml:"info"`
	Reg  []ResReg `yaml:"regression"`
}

type InfoReg struct {
	Image       string `yaml:"image"`
	CpuModel    string `yaml:"cpu_model"`
	NodeType    string `yaml:"node_type,omitempty"`
	ROI         string `yaml:"max_roi"`
	LastUpdated string `yaml:"last_updated"`
}

/* type InfoReg struct {
	ResRegs []ResReg `yaml:"name"`
} */

type ResReg struct {
	Name            string           `yaml:"name"`
	R2              float64          `yaml:"R²"`
	VarianceObs     float64          `yaml:"variance_obs"`
	VariancePred    float64          `yaml:"variance_pred"`
	N               int              `yaml:"N_obs"`
	Formula         string           `yaml:"formula"`
	Offset          float64          `yaml:"offset"`
	Unit            string           `yaml:"unit"`
	Limit           float64          `yaml:"limit,omitempty"`
	PredictorCoeffs []PredictorCoeff `yaml:"predictor"`
}

type PredictorCoeff struct {
	Name   string    `yaml:"name"`
	Coeffs []float64 `yaml:"coeffs_deg,flow"`
}

/* ---
- name: resource-control-sample
  info:
    image: ghcr.io/arnaudlemaignen/resource-control-sample:latest
	cpu_model: "Intel(R) Core(TM) i5-5200U CPU @ 2.20GHz"
	node_type: r5.xlarge
	max_roi: 1h0m0s
  regression:
  - name: cpu
    r²: 0.9500
	variance_obs: 3950
	variance_pred: 3773
	n: 51
	formula: 2.4378 + image_pixels_M*1.2599 + (image_pixels_M)^2*0.1
	offset: 2.4378
	predictor:
	- name: image_pixels_M
      coeffs_deg: [1.2599,0.1,0,0]
  - name: mem
    R²: 0.9500
	variance_obs: 3950
	variance_prde: 3773
	n: 51
	formula: 2.4378 + image_pixels_M*1.2599
	predictor:
	- name: image_pixels_M
      predictor_coeffs_deg: [1.2599,0,0,0]
*/

func OpenRegressions(filename string) []ContainerReg {
	err := utils.CheckFile(filename)
	if err != nil {
		log.Error(err)
	}
	byteValue, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Error(err)
	}
	var containers []ContainerReg
	yaml.Unmarshal(byteValue, &containers)
	if len(containers) == 0 {
		log.Info(filename, " is empty or not well formated")
	}

	return containers
}

func WriteBestSoFarRegFromYaml(filename string, containers []ContainerReg) {
	err := utils.CheckFile(filename)
	if err != nil {
		log.Error(err)
	}
	/* 	file, err := ioutil.ReadFile(filename)
	   	if err != nil {
	   		log.Error(err)
	   	} */

	// Preparing the data to be marshalled and written.
	dataBytes, err := yaml.Marshal(containers)
	if err != nil {
		log.Error(err)
	}
	err = ioutil.WriteFile(filename, dataBytes, 0644)
	if err != nil {
		log.Error(err)
	}
}
