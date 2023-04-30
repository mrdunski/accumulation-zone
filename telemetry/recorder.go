package telemetry

import (
	"context"
	"sync"
	"time"
)

type Recorder interface {
	Record()
}

type ContinuousRecorder interface {
	Recorder
	ContinuousRecord(ctx context.Context)
}

type multiRecorder struct {
	recorders []Recorder
	config    TeleConfig
	mutex     sync.Mutex
}

func (r *multiRecorder) Record() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	for _, recorder := range r.recorders {
		recorder.Record()
	}
}

func (r *multiRecorder) ContinuousRecord(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(r.config.PushGatewayUpdateRate):
			r.Record()
		}
	}
}
