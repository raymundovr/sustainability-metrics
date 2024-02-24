package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

func main() {
	prometheusURL := flag.String("prometheus-url", "http://localhost:9090", "Prometheus URL")
	duration := flag.Int("duration", 15, "Duration in minutes")
	testType := flag.String("test-type", "idle", "The type of test that you're observing, ex: idle, stress-test, etc")
	project := flag.String("project", "", "Project's name")
	projectNamespace := flag.String("namespace", "", "The Namespace where the observed project resides")
	node := flag.String("node", "", "The node where the Project is running")

	flag.Parse()

	provider, err := NewPrometheusProvider(*prometheusURL)
	if err != nil {
		log.Fatal("Cannot create Provider: ", err.Error())
	}

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
			Query:        fmt.Sprintf(`instance:node_cpu_utilisation:rate5m{job="node-exporter", instance="%s"} != 0`, *node),
			WatchMetrics: []string{"instance"},
		},
	}

	fmt.Println("project_name:", *project)
	fmt.Println("project_namespace:", *projectNamespace)
	fmt.Println("node:", *node)
	fmt.Println("test_type:", *testType)
	var m sync.Mutex // avoid mixing results when printing

	for i := 0; i < *duration; i++ {
		now := time.Now()
		timeRange := timerange{
			Start: now.Add(-time.Minute),
			End:   now,
		}
		for _, q := range queries {
			go func(query query) {
				records, err := provider.PerformQuery(query, timeRange)
				m.Lock()
				defer m.Unlock()
				fmt.Println("query:", query.Id)
				fmt.Println("query_time_start:", timeRange.Start.Unix())
				fmt.Println("query_time_end:", timeRange.End.Unix())
				fmt.Println("results:")
				if err != nil {
					fmt.Printf("Cannot get results for query %s: %s\n", query.Id, err.Error())
					return
				}
				header := append(query.WatchMetrics, "timestamp", "value")
				fmt.Println(strings.Join(header, ","))
				for _, line := range records {
					fmt.Println(strings.Join(line, ","))
				}
			}(q)
		}
		time.Sleep(1 * time.Minute)
	}
}


