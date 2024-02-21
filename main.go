package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type records [][]string

type query struct {
	Id           string
	Query        string
	WatchMetrics []string
}

type timerange struct {
	Start time.Time
	End   time.Time
}

func main() {
	prometheusURL := flag.String("prometheus-url", "http://localhost:9090", "Prometheus URL")
	duration := flag.Int("duration", 15, "Duration in minutes")
	testType := flag.String("test-type", "idle", "The type of test that you're observing, ex: idle, stress-test, etc")
	project := flag.String("project", "", "Project's name")
	projectNamespace := flag.String("namespace", "", "The Namespace where the observed project resides")
	node := flag.String("node", "", "The node where the Project is running")

	flag.Parse()

	client, err := api.NewClient(api.Config{
		Address: *prometheusURL,
	})
	if err != nil {
		fmt.Println("Error creating client:", err)
		return
	}

	queryAPI := v1.NewAPI(client)

	queries := []query{
		{
			Id:           "kepler_dram",
			Query:        fmt.Sprintf(`sum by (pod_name, container_namespace) (irate(kepler_container_dram_joules_total{container_namespace=~"%s",pod_name=~".*"}[1m]))`, *projectNamespace),
			WatchMetrics: []string{"container_namespace", "pod_name"},
		},
		{
			Id:           "kepler_package",
			Query:        fmt.Sprintf(`sum by (pod_name, container_namespace) (irate(kepler_container_package_joules_total{container_namespace=~"%s",pod_name=~".*"}[1m]))`, *projectNamespace),
			WatchMetrics: []string{"container_namespace", "pod_name"},
		},
		{
			Id:           "cpu_utilization_node",
			Query:        fmt.Sprintf(`instance:node_cpu_utilisation:rate5m{job="node-exporter", instance="%s", cluster=""} != 0`, *node),
			WatchMetrics: []string{"instance"},
		},
	}

	fmt.Println("project_name:", *project)
	fmt.Println("project_namespace:", *projectNamespace)
	fmt.Println("node:", *node)
	fmt.Println("test_type:", *testType)

	for i := 0; i < *duration; i++ {
		for _, q := range queries {
			go func(query query) {
				now := time.Now()
				timeRange := timerange{
					Start: now.Add(-time.Minute),
					End:   now,
				}
				records, err := performQuery(queryAPI, query, timeRange)
				if err != nil {
					fmt.Printf("Cannot get results for query %s: %s\n", query.Id, err.Error())
					return
				}
				fmt.Println("query:", query.Id)
				fmt.Println("query_time_start:", timeRange.Start.Unix())
				fmt.Println("query_time_end:", timeRange.End.Unix())
				fmt.Println("results:")
				for _, line := range records {
					fmt.Println(line)
				}
			}(q)
		}
		time.Sleep(1 * time.Minute)
	}
}

func performQuery(queryAPI v1.API, query query, timeRange timerange) (records, error) {
	result, warnings, err := queryAPI.QueryRange(context.Background(), query.Query, v1.Range{
		Start: timeRange.Start,
		End:   timeRange.End,
		Step:  15 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	if len(warnings) > 0 {
		fmt.Println("\tWarnings received:", warnings)
	}

	// Print query result
	matrix, ok := result.(model.Matrix)
	if !ok {
		return nil, errors.New("result is not a matrix")
	}
	// namespace, pod, time, value
	records := [][]string{}
	for _, series := range matrix {
		labelValues := []string{}
		for _, label := range query.WatchMetrics {
			val, ok := series.Metric[model.LabelName(label)]
			if !ok {
				fmt.Println("\tWarn: Label", label, "does not exist for", query.Id)
			}
			labelValues = append(labelValues, string(val))
		}
		for _, sample := range series.Values {
			record := append(labelValues, sample.Timestamp.String(), sample.Value.String())
			records = append(records, record)
		}
	}

	return records, nil
}
