package main

import (
	"flag"
	"log"
	"time"

	"contrib.go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

var (
	jaegerServer     = flag.String("jaegerServer", "http://10.10.40.2:30932/api/traces", "jaeger http url")
	jaegerServerName = flag.String("jaegerServerName", "event-adapter", "jager service name. ")
)

func init() {

	// init trace
	if *jaegerServer != "" {
		exp, err := jaeger.NewExporter(jaeger.Options{
			CollectorEndpoint: *jaegerServer,
			Process: jaeger.Process{
				ServiceName: *jaegerServerName,
			},
		})
		if err != nil {
			log.Fatal("get exporter failed: ", err)
		}
		trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
		trace.RegisterExporter(exp)
		view.SetReportingPeriod(1 * time.Second)
	}

}
