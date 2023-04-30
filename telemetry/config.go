package telemetry

import (
	"time"

	"github.com/google/uuid"
)

type TeleConfig struct {
	PushTelemetry         bool          `help:"Push telemetry to Push Gateway" env:"PUSH_TELEMETRY" optional:""`
	PrintTelemetry        bool          `help:"Logs telemetry as info" env:"PRINT_TELEMETRY" optional:""`
	PushGatewayUrl        string        `help:"Address of Push Gateway" env:"PUSH_GATEWAY_URL" default:"http://localhost:9091"`
	TelemetryJobName      string        `help:"Name for the job" env:"TELEMETRY_JOB_NAME" default:"az"`
	TelemetryJobId        string        `help:"Job identifier (random id by default)" env:"TELEMETRY_JOB_ID" optional:""`
	PushGatewayUpdateRate time.Duration `help:"Defines how often push will be made" env:"PUSH_GATEWAY_UPDATE_RATE" default:"60s"`
}

func (c TeleConfig) NewRecorder() ContinuousRecorder {
	cfg := c
	if cfg.TelemetryJobId == "" {
		cfg.TelemetryJobId = uuid.NewString()
	}

	mr := &multiRecorder{config: cfg}

	if cfg.PushTelemetry {
		mr.recorders = append(mr.recorders, pusher{config: cfg})
	}

	if cfg.PrintTelemetry {
		mr.recorders = append(mr.recorders, printer{config: cfg})
	}

	return mr
}
