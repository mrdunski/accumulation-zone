package telemetry

import (
	"github.com/mrdunski/accumulation-zone/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

type pusher struct {
	config TeleConfig
}

func (p pusher) Record() {
	log := logger.WithComponent("telemetry")

	if err := push.New(p.config.PushGatewayUrl, p.config.TelemetryJobName).
		Gatherer(prometheus.DefaultGatherer).
		Grouping("id", p.config.TelemetryJobId).
		Push(); err != nil {
		log.WithError(err).Error("Could not push metrics to Push gateway.")
	}
}
