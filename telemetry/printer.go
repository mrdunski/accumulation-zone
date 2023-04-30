package telemetry

import (
	"github.com/mrdunski/accumulation-zone/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type printer struct {
	config TeleConfig
}

func (p printer) Record() {
	log := logger.WithComponent("telemetry")

	metrics, err := prometheus.DefaultGatherer.Gather()

	if err != nil {
		log.WithError(err).Error("Failed to gather metrics")
		return
	}

	for _, metricFamily := range metrics {
		for _, metric := range metricFamily.Metric {
			log.WithFields(logrus.Fields{
				"name":   metricFamily.GetName(),
				"help":   metricFamily.GetHelp(),
				"labels": metric.GetLabel(),
			}).Infof("metric: %v", metric)
		}
	}
}
