package utils

import (
	"io/ioutil"
	"os"

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

func CheckFile(filename string) error {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		_, err := os.Create(filename)
		if err != nil {
			return err
		}
	}
	return nil
}
