package utils

import (
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
)

// MustGet will return the env or panic if it is not present
func GetStringEnv(k string, d string) string {
	v := os.Getenv(k)
	if v == "" {
		// log.MissingArg(k)
		log.Warn("ENV missing, key: ", k, " will use default ", d)
		return d
	}
	return v
}

// MustGetBool will return the env as boolean or panic if it is not present
func GetBoolEnv(k string, d bool) bool {
	v := os.Getenv(k)
	if v == "" {
		// log.MissingArg(k)
		log.Warn("ENV missing, key: ", k, " will use default ", d)
		return d
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		// log.MissingArg(k)
		log.Warn("ENV err: ["+k+"]"+err.Error(), " will use default ", d)
		return d
	}
	return b
}

// MustGetInt32 will return the env as int32 or panic if it is not present
func GetIntEnv(k string, d int) int {
	v := os.Getenv(k)
	if v == "" {
		// log.MissingArg(k)
		log.Warn("ENV missing, key: "+k, " will use default ", d)
		return d
	}
	i, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		// log.MissingArg(k)
		log.Warn("ENV err: ["+k+"]"+err.Error(), " will use default ", d)
		return d
	}
	return int(i)
}

// MustGetInt64 will return the env as int64 or panic if it is not present
func GetInt64Env(k string, d int64) int64 {
	v := os.Getenv(k)
	if v == "" {
		// log.MissingArg(k)
		log.Warn("ENV missing, key: ", k, " will use default ", d)
		return d
	}
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		// log.MissingArg(k)
		log.Warn("ENV err: ["+k+"]"+err.Error(), " will use default ", d)
		return d
	}
	return i
}
