package main

import (
	"context"
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/mrdunski/accumulation-zone/cmd/changes/commit"
	"github.com/mrdunski/accumulation-zone/cmd/changes/ls"
	"github.com/mrdunski/accumulation-zone/cmd/changes/upload"
	"github.com/mrdunski/accumulation-zone/cmd/inventory"
	"github.com/mrdunski/accumulation-zone/cmd/restore"
	"github.com/mrdunski/accumulation-zone/logger"
	"github.com/mrdunski/accumulation-zone/telemetry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	startGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: telemetry.Namespace,
		Name:      "job_start_time",
	})

	endGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: telemetry.Namespace,
		Name:      "job_end_time",
	})

	executionTime = promauto.NewSummary(prometheus.SummaryOpts{
		Namespace: telemetry.Namespace,
		Name:      "job_execution_time",
	})

	resultCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: telemetry.Namespace,
		Name:      "job_result",
	}, []string{"result"})

	successCounter = resultCounter.With(prometheus.Labels{"result": "success"})
	failureCounter = resultCounter.With(prometheus.Labels{"result": "failure"})
	startedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: telemetry.Namespace,
		Name:      "job_started",
	})
)

type CommandInput struct {
	logger.LogConfig
	telemetry.TeleConfig

	Recover struct {
		Index restore.IndexCmd `cmd:"" help:"Recovers index file from glacier."`
		Data  restore.DataCmd  `cmd:"" help:"Recovers data from glacier."`
		All   restore.AllCmd   `cmd:"" help:"Recovers index and data from glacier."`
	} `cmd:"" help:"Various backup recovery options." group:"Recover"`

	Changes struct {
		Upload upload.Cmd `cmd:"" help:"Uploads all changes to AWS vault and commits them as processed." group:"Backup"`
		Ls     ls.Cmd     `cmd:"" help:"List changes in the directory."`
		Commit commit.Cmd `cmd:"" help:"DANGER: marks all detected changes as processed and it won't be processed in the future."`
	} `cmd:"" help:"Changes management." group:"Manage Changes"`

	Inventory struct {
		Retrieve inventory.RetrieveCmd `cmd:"" help:"Starts retrieval job for inventory."`
		Print    inventory.PrintCmd    `cmd:"" help:"Awaits latest job completion and prints inventory."`
		Purge    inventory.PurgeCmd    `cmd:"" help:"DANGER: removes all archives from vault. It requires existing retrieval job (in progress or completed)."`
	} `cmd:"" help:"Admin operations on glacier inventory." group:"Manage Glacier Inventory"`
}

func main() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	timer := prometheus.NewTimer(executionTime)

	startGauge.SetToCurrentTime()
	startedCounter.Inc()
	var input = CommandInput{}
	kongCtx := kong.Parse(&input)
	input.LogConfig.InitLogger(kongCtx)
	reportJobInfo(kongCtx)

	tele := input.TeleConfig.NewRecorder()
	tele.Record()
	go tele.ContinuousRecord(ctx)

	defer func() {
		endGauge.SetToCurrentTime()
		cancelFunc()
		if r := recover(); r != nil {
			failureCounter.Inc()
			tele.Record()
			panic(r)
		}
		successCounter.Inc()
		tele.Record()
	}()
	defer timer.ObserveDuration()

	err := kongCtx.Run()
	if err != nil {
		logger.Get().WithError(err).Fatalf("Command failed")
	}
}

func reportJobInfo(kongCtx *kong.Context) {
	metric := prometheus.MustNewConstMetric(
		prometheus.NewDesc(fmt.Sprintf("%s_%s", telemetry.Namespace, "job_info"), "details about job execution", []string{}, prometheus.Labels{
			"command": kongCtx.Command(),
		}),
		prometheus.GaugeValue,
		1)

	c := collectorWrapper{Metric: metric}

	prometheus.DefaultRegisterer.MustRegister(c)

}

type collectorWrapper struct {
	prometheus.Metric
}

func (c collectorWrapper) Describe(descs chan<- *prometheus.Desc) {
	descs <- c.Desc()
}

func (c collectorWrapper) Collect(metrics chan<- prometheus.Metric) {
	metrics <- c.Metric
}
