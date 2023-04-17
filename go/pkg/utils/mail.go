package utils

import (
	"crypto/tls"
	"io/ioutil"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	gomail "gopkg.in/gomail.v2"
)

type FilesToAttach struct {
	Dir string
	Ext string
}

// const outPathTarball = types.TMP_PATH + "dmilib.tar.gz"

// https://pkg.go.dev/gopkg.in/gomail.v2#Dialer
func Send() bool {
	m := gomail.NewMessage()
	from := GetStringEnv("MAIL_ORIGINATOR", "")
	to := GetStringEnv("MAIL_TO_AGGR", "")
	hello := GetStringEnv("MAIL_SMTP_HELLO", "localhost")
	host := GetStringEnv("MAIL_SMTP_HOST", "localhost")
	port := GetIntEnv("MAIL_SMTP_PORT", 25)
	user := GetStringEnv("MAIL_SMTP_USER", "")
	pwd := GetStringEnv("MAIL_SMTP_PWD", "")
	subj := time.Now().Format("20060102") + "_" + GetStringEnv("ROUTE_SUBDOMAIN", "domain") + "_" + GetStringEnv("NAMESPACE", "ns") + "_ResourceModelExporter"
	body := "This mail should contains 5 yaml files to be extracted"
	toAttach := []FilesToAttach{
		{Dir: RES_PATH, Ext: ".yaml"},
		{Dir: OUT_PATH, Ext: ".yaml"},
		{Dir: LOG_PATH, Ext: ".log"}}

	m.SetHeader("From", from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subj)
	m.SetBody("text/plain", body)

	attachments := findAttachmentFiles(toAttach)
	for _, attachment := range attachments {
		m.Attach(attachment)
	}
	d := gomail.NewDialer(host, port, user, pwd)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	d.LocalName = hello
	log.Debug("Before sending : host/", d.Host, " port/", d.Port, " user/", d.Username, " auth/", d.Auth, " ssl/", d.SSL, " hello/", d.LocalName)
	d.SSL = false

	if err := d.DialAndSend(m); err != nil {
		log.Error("Can not dial or send email ", subj, " err ", err)
		log.Error("Additional info : host/", d.Host, " port/", d.Port, " user/", d.Username, " auth/", d.Auth, " ssl/", d.SSL, " hello/", d.LocalName)
		return false
	}

	return true
}

func findAttachmentFiles(toAttach []FilesToAttach) []string {
	var result []string
	for _, attach := range toAttach {
		files, err := ioutil.ReadDir(attach.Dir)
		if err != nil {
			log.Error("Error reading dir ", attach.Dir, " err ", err)
		}

		for _, file := range files {
			if filepath.Ext(file.Name()) == attach.Ext {
				result = append(result, filepath.Join(attach.Dir, file.Name()))
			}
		}
	}

	return result
}
