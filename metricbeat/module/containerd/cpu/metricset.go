// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package cpu

import (
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

// Metricset for apiserver is a prometheus based metricset
type metricset struct {
	mb.BaseMetricSet
	prometheusClient   prometheus.Prometheus
	prometheusMappings *prometheus.MetricsMapping
}

var _ mb.ReportingMetricSetV2Error = (*metricset)(nil)

// getMetricsetFactory as required by` mb.Registry.MustAddMetricSet`
func getMetricsetFactory(prometheusMappings *prometheus.MetricsMapping) mb.MetricSetFactory {
	return func(base mb.BaseMetricSet) (mb.MetricSet, error) {
		pc, err := prometheus.NewPrometheusClient(base)
		if err != nil {
			return nil, err
		}
		return &metricset{
			BaseMetricSet:      base,
			prometheusClient:   pc,
			prometheusMappings: prometheusMappings,
		}, nil
	}
}

// Fetch gathers information from the containerd and reports events with this information.
func (m *metricset) Fetch(reporter mb.ReporterV2) error {
	events, err := m.prometheusClient.GetProcessedMetrics(m.prometheusMappings)
	if err != nil {
		return errors.Wrap(err, "error getting metrics")
	}
	m.Logger().Debugf("Events from containerd are %+v", events)
	for _, event := range events {
		// applying ECS to kubernetes.container.id in the form <container.runtime>://<container.id>
		// copy to ECS fields the kubernetes.container.image, kubernetes.container.name
		containerFields := common.MapStr{}
		if containerID, ok := event["id"]; ok {
			// we don't expect errors here, but if any we would obtain an
			// empty string
			cID := (containerID).(string)
			containerFields.Put("id", cID)
			event.Delete("id")
		}

		e, err := util.CreateEvent(event, "containerd.cpu")
		if err != nil {
			m.Logger().Error(err)
		}

		if len(containerFields) > 0 {
			if e.RootFields != nil {
				e.RootFields.DeepUpdate(common.MapStr{
					"container": containerFields,
				})
			} else {
				e.RootFields = common.MapStr{
					"container": containerFields,
				}
			}
		}

		if reported := reporter.Event(e); !reported {
			return nil
		}
	}

	return nil
}