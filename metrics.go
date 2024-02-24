package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
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

type Provider interface {
	PerformQuery(query query, timeRange timerange) (records, error)
}

func PrintMetrics(queries []query, provider Provider, duration int) {
	var m sync.Mutex      // avoid mixing results when printing
	var wgq sync.WaitGroup // wait for all queries

	for i := 0; i < duration; i++ {
		time.Sleep(1 * time.Minute)
		now := time.Now()
		timeRange := timerange{
			Start: now.Add(-time.Minute),
			End:   now,
		}
		for _, q := range queries {
			wgq.Add(1)
			go func(query query) {
				defer wgq.Done()
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
			wgq.Wait()
		}
	}
}
