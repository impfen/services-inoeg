// Kiebitz - Privacy-Friendly Appointment Scheduling
// Copyright (C) 2021-2021 The Kiebitz Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version. Additional terms
// as defined in section 7 of the license (e.g. regarding attribution)
// are specified at https://kiebitz.eu/en/docs/open-source/additional-terms.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package servers

import (
	"github.com/impfen/services-inoeg"
	"sort"
	"strings"
	"time"
)

// public endpoint
func (c *Appointments) getStats(context services.Context, params *services.GetStatsParams) services.Response {

	if c.meter == nil {
		return context.InternalError()
	}

	toTime := time.Now().UTC().UnixNano()

	var metrics []*services.Metric
	var err error

	if params.N != nil {
		metrics, err = c.meter.N(params.ID, toTime, *params.N, params.Name, params.Type)
	} else {
		metrics, err = c.meter.Range(params.ID, params.From.UnixNano(), params.To.UnixNano(), params.Name, params.Type)
	}

	if err != nil {
		services.Log.Error(err)
		return context.InternalError()
	}

	values := make([]*services.StatsValue, 0)

addMetric:
	for _, metric := range metrics {
		if params.Metric != "" && metric.Name != params.Metric {
			continue
		}
		if metric.Name[0] == '_' {
			// we skip internal metrics (which start with a '_')
			continue
		}

		if params.Filter != nil {
			for k, v := range params.Filter {
				// if v is nil we only return metrics without a value for the given key
				if v == nil {
					if _, ok := metric.Data[k]; ok {
						continue addMetric
					}
				} else if dv, ok := metric.Data[k]; !ok || dv != v {
					// filter value is missing or does not match
					continue addMetric
				}
			}
		}

		values = append(values, &services.StatsValue{
			From:  time.Unix(metric.TimeWindow.From/1e9, metric.TimeWindow.From%1e9).UTC(),
			To:    time.Unix(metric.TimeWindow.To/1e9, metric.TimeWindow.From%1e9).UTC(),
			Name:  metric.Name,
			Value: metric.Value,
			Data:  metric.Data,
		})
	}

	// we store the statistics
	sortableValues := Values{values: values}
	sort.Sort(sortableValues)

	return context.Result(values)
}

type Values struct {
	values []*services.StatsValue
}

func (f Values) Len() int {
	return len(f.values)
}

func (f Values) Less(i, j int) bool {
	r := (f.values[i].From).Sub(f.values[j].From)
	if r < 0 {
		return true
	}
	// if the from times match we compare the names
	if r == 0 {
		if strings.Compare(f.values[i].Name, f.values[j].Name) < 0 {
			return true
		}
	}
	return false
}

func (f Values) Swap(i, j int) {
	f.values[i], f.values[j] = f.values[j], f.values[i]

}
