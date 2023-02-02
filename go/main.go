package main

import (
	"flag"
	"net/http"
	"os"
	"resource-model-exporter/pkg/collector"
	"resource-model-exporter/pkg/types"

	"github.com/joho/godotenv"

	time "github.com/hyperjumptech/jiffy"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	version          = "v0.01"
	listenAddress    = flag.String("web.listen-address", ":9901", "Address to listen on for telemetry")
	metricsPath      = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics")
	outputDir        = "output/"
	resYmlPredictors = "resources/predictors.yml"
	resYmlObserved   = "resources/observed.yml"
	resYmlInfo       = "resources/info.yml"
	resYmlLimit      = "resources/limits.yml"
	outYmlReg        = outputDir + "regressions.yml"
)

//Readiness message
func Ready() string {
	return "Resource Model Exporter is ready"
}

func Init() *collector.Exporter {
	flag.Parse()

	err := godotenv.Load(".env")
	if err != nil {
		log.Info(".env file absent, assume env variables are set.")
	}

	promEndpoint := os.Getenv("PROM_ENDPOINT")
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
	exporter := collector.NewExporter(promURL, version, outYmlReg, timeMaxRoi, timeInterval, predictors, observed, info, limits, regs)
	prometheus.MustRegister(exporter)

	// test()
	return exporter
}

// func test() {
// 	r := new(regression.Regression)
// 	r.SetObserved("Murders per annum per 1,000,000 inhabitants")
// 	r.SetVar(0, "Inhabitants")
// 	r.SetVar(1, "Percent with incomes below $5000")
// 	r.SetVar(2, "Percent unemployed")
// 	r.Train(
// 		regression.DataPoint(11.2, []float64{587000, 16.5, 6.2}),
// 		regression.DataPoint(13.4, []float64{643000, 20.5, 6.4}),
// 		regression.DataPoint(40.7, []float64{635000, 26.3, 9.3}),
// 		regression.DataPoint(5.3, []float64{692000, 16.5, 5.3}),
// 		regression.DataPoint(24.8, []float64{1248000, 19.2, 7.3}),
// 		regression.DataPoint(12.7, []float64{643000, 16.5, 5.9}),
// 		regression.DataPoint(20.9, []float64{1964000, 20.2, 6.4}),
// 		regression.DataPoint(35.7, []float64{1531000, 21.3, 7.6}),
// 		regression.DataPoint(8.7, []float64{713000, 17.2, 4.9}),
// 		regression.DataPoint(9.6, []float64{749000, 14.3, 6.4}),
// 		regression.DataPoint(14.5, []float64{7895000, 18.1, 6}),
// 		regression.DataPoint(26.9, []float64{762000, 23.1, 7.4}),
// 		regression.DataPoint(15.7, []float64{2793000, 19.1, 5.8}),
// 		regression.DataPoint(36.2, []float64{741000, 24.7, 8.6}),
// 		regression.DataPoint(18.1, []float64{625000, 18.6, 6.5}),
// 		regression.DataPoint(28.9, []float64{854000, 24.9, 8.3}),
// 		regression.DataPoint(14.9, []float64{716000, 17.9, 6.7}),
// 		regression.DataPoint(25.8, []float64{921000, 22.4, 8.6}),
// 		regression.DataPoint(21.7, []float64{595000, 20.2, 8.4}),
// 		regression.DataPoint(25.7, []float64{3353000, 16.9, 6.7}),
// 	)
// 	r.Run()
//
// 	fmt.Printf("Regression formula:\n%v\n", r.Formula)
// 	fmt.Printf("Regression:\n%s\n", r)
// }

func main() {
	log.Info("Starting resource model exporter")
	Init()

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
	log.Info("Listening on port " + *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
