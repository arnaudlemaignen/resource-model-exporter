package main

import (
	"flag"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"resource-model-exporter/pkg/collector"
	"resource-model-exporter/pkg/types"
	"resource-model-exporter/pkg/utils"
	"syscall"
	timestd "time"

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"

	time "github.com/hyperjumptech/jiffy"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	version          = "v0.01"
	debug            = flag.Bool("debug", false, "Debug mode default false")
	email            = flag.Bool("email", true, "Send Mails mode default true")
	listenAddress    = flag.String("web.listen-address", ":9901", "Address to listen on for telemetry")
	metricsPath      = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics")
	outputDir        = utils.OUT_PATH
	logDir           = utils.LOG_PATH
	resDir           = utils.RES_PATH
	resYmlPredictors = resDir + "/predictors.yml"
	resYmlObserved   = resDir + "/observed.yml"
	resYmlInfo       = resDir + "/info.yml"
	resYmlLimit      = resDir + "/limits.yml"
	outYmlReg        = outputDir + "/regressions.yml"
)

// Readiness message
func Ready() string {
	return "Resource Model Exporter is ready"
}

func Init() *collector.Exporter {
	flag.Parse()
	manageLogger()

	err := godotenv.Load(".env")
	if err != nil {
		log.Warn(".env file absent, assume env variables are set.")
	}

	promEndpoint := os.Getenv("PROMETHEUS_ENDPOINT")
	promUser := os.Getenv("PROMETHEUS_AUTH_USER")
	promPwd := os.Getenv("PROMETHEUS_AUTH_PWD")
	promLogin := ""
	if promUser == "" || promPwd == "" {
		log.Info("PROMETHEUS_AUTH_USER and or PROMETHEUS_AUTH_PWD were not set, will not use basic auth.")
	} else {
		promLogin = promUser + ":" + promPwd
	}
	maxRoi := os.Getenv("REGRESSION_MAX_ROI")
	interval := os.Getenv("SAMPLING_INTERVAL")

	// http://user:pass@localhost/ to use basic auth
	promURL := "http://" + promEndpoint
	if len(promLogin) > 0 {
		promURL = "http://" + promLogin + "@" + promEndpoint
	}
	log.Info("prometheus URL         => ", promURL)
	log.Info("Max Region Of Interest => ", maxRoi)
	log.Info("Sampling Interval      => ", interval)

	os.MkdirAll(outputDir, os.ModePerm)
	//populate services maps
	predictors := types.OpenPredictors(resYmlPredictors)
	observed := types.OpenObserved(resYmlObserved)
	info := types.OpenInfo(resYmlInfo)
	limits := types.OpenLimits(resYmlLimit)
	regs := types.OpenRegressions(outYmlReg)
	//1. check that at least container is provided
	//2. check that all $vars from dim input factor queries can be reconciled

	//3. check interval and minRoi/maxRoi are timestep
	//time.ParseDuration does not supprt d w :(
	timeMaxRoi, errMaxRoi := time.DurationOf(maxRoi)
	if errMaxRoi != nil {
		log.Fatal("Invalid time duration for maxRoi ", maxRoi, " err: ", errMaxRoi)
	}
	timeInterval, errInterval := time.DurationOf(interval)
	if errInterval != nil {
		log.Fatal("Invalid time duration for interval ", interval, " err: ", errInterval)
	}

	//Registering Exporter
	exporter := collector.NewExporter(promURL, version, resYmlPredictors, resYmlObserved, resYmlInfo, resYmlLimit, outYmlReg, timeMaxRoi, timeInterval, predictors, observed, info, limits, regs)
	prometheus.MustRegister(exporter)

	return exporter
}

func manageLogger() {
	log.SetOutput(os.Stdout)

	//activate debug mode otherwise Info as log level
	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	os.MkdirAll(logDir, os.ModePerm)
	logFile := filepath.Join(logDir, timestd.Now().Format("200701121604")+"_ResourceModelExporter.log")
	f, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		log.Warn("Issue with log file ", logFile)
	}
	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)
}

func main() {
	log.Info("Starting resource model exporter")
	exporter := Init()

	//This section will start the HTTP server and expose
	//any metrics on the /metrics endpoint.
	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Resource Model Exporter ` + version + `</title></head>
             <body>
             <h1>` + Ready() + `'</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})

	// Launch a cron that will send an email once a day at midnight
	c := cron.New()
	c.AddFunc("@daily", func() { sendEmails(*email) })
	c.Start()

	//To catch SIGHUP signal for reloading conf
	//https://rossedman.io/blog/computers/hot-reload-sighup-with-go/
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP)

	go func() {
		log.Info("Listening on port " + *listenAddress)
		log.Fatal(http.ListenAndServe(*listenAddress, nil))
		c.Stop()
	}()

	for range sigs {
		log.Warn("HOT RELOAD")
		exporter.ReloadYamlConfig()
	}
}
