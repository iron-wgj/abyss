package main

import (
	"fmt"
	"log"
	"strings"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"wanggj.com/abyss/module"
)

var (
	influxdb_url    = "https://us-east-1-1.aws.cloud2.influxdata.com"
	influxdb_token  = "NWJxYmxLdXSrr_DjuhDk3gabymzNjNGKNX0x2j0d8YyM5sacAL4m8okBRv10YnLp-0cZI_ftVpEuGDMWjQu7LQ=="
	influxdb_org    = "abyss"
	influxdb_bucket = "write_test"
)

// This file contains functions used to write MetricFamilys
// into different file or databahse

// InfluxWrite is used to write into influxdb
// token: NWJxYmxLdXSrr_DjuhDk3gabymzNjNGKNX0x2j0d8YyM5sacAL4m8okBRv10YnLp-0cZI_ftVpEuGDMWjQu7LQ==
func InfluxWrite(logger *log.Logger, dataCh <-chan map[int][]*module.MetricFamily) {
	client := influxdb2.NewClient(influxdb_url, influxdb_token)
	writeAPI := client.WriteAPI(influxdb_org, influxdb_bucket)
	errorCh := writeAPI.Errors()

	for data := range dataCh {
		go influxWriteLines(writeAPI, data)
	}

	for err := range errorCh {
		fmt.Println(err)
	}

	client.Close()
}

func influxWriteLines(w_api api.WriteAPI, data map[int][]*module.MetricFamily) {
	// write directly line protocol
	for _, mfs := range data {
		for _, mf := range mfs {
			lines := metricFamily2line(mf)
			for _, line := range lines {
				w_api.WriteRecord(line)
			}
		}
	}
	//w_api.Flush()
}

func metricFamily2line(mf *module.MetricFamily) []string {
	result := []string{}
	measurement := mf.GetName()
	for _, m := range mf.Metric {
		line := metric2line(measurement, m, mf.GetType())
		if line == "" {
			continue
		}
		result = append(result, line)
		fmt.Println(line)
	}
	return result
}

func metric2line(measurement string, m *module.Metric, mfType module.MetricType) string {
	labels := m.GetLabel()
	lptags := make([]string, 0, len(labels))
	for _, l := range labels {
		lptags = append(lptags, fmt.Sprintf("%s=%s", l.GetName(), l.GetValue()))
	}
	timestamp := m.GetTimestamp().AsTime().UnixNano()
	var lpvals []string
	switch mfType {
	case module.MetricType_COUNTER:
		if m.Counter == nil {
			return ""
		}
		lpvals = []string{fmt.Sprintf("%s=%f", "counter", m.Counter.GetValue())}
	case module.MetricType_GAUGE:
		if m.Gauge == nil {
			return ""
		}
		lpvals = []string{fmt.Sprintf(
			"%s=%f", "gauge", m.Gauge.GetValue(),
		)}
	case module.MetricType_EVENT:
		if m.Event == nil {
			return ""
		}
		lpvals = []string{
			fmt.Sprintf("%s=%f", "event_value", m.Event.GetValue()),
		}
		timestamp = m.Event.GetTimestamp().AsTime().UnixNano()
	case module.MetricType_SUMMARY:
		if m.Summary == nil {
			return ""
		}
		qua := m.Summary.GetQuantile()
		lpvals = make([]string, 0, len(qua))
		for _, q := range qua {
			lpvals = append(lpvals, fmt.Sprintf(
				"quantile_%2f=%f", q.GetQuantile(), q.GetValue(),
			))
		}
	}

	return fmt.Sprintf(
		"%s,%s %s %d",
		measurement,
		strings.Join(lptags, ","),
		strings.Join(lpvals, ","),
		timestamp,
	)

}
