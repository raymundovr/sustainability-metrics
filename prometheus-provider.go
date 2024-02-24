package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"github.com/prometheus/common/model"
)

type PrometheusProvider struct {
	queryAPI v1.API
}

func NewPrometheusProvider(url string) (*PrometheusProvider, error) {
	client, err := api.NewClient(api.Config{
		Address: url,
	})
	if err != nil {
		return nil, err
	}

	queryAPI := v1.NewAPI(client)
	return &PrometheusProvider{
		queryAPI: queryAPI,
	}, nil
}

func (p *PrometheusProvider) PerformQuery(query query, timeRange timerange) (records, error) {
	result, warnings, err := p.queryAPI.QueryRange(context.Background(), query.Query, v1.Range{
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
