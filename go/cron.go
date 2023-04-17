package main

import (
	"resource-model-exporter/pkg/utils"

	log "github.com/sirupsen/logrus"
)

func sendEmails(toSend bool) {
	if toSend {
		success := utils.Send()
		if success {
			log.Info("Resource Model Exporter daily email was sent")
		}
	} else {
		log.Info("Sending mail is disabled")
	}
}
