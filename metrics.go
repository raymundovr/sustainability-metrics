package main

import "time"

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