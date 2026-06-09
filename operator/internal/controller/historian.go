package controller

import (
	"context"
	"time"
)

// HistorianSink is the observability extension point for the Tag controller.
// When HistorianEnabled is true on a Tag, the controller calls Record after
// every successful read.
//
// Plug in the InfluxDB implementation once infra is live:
//
//	import historian "github.com/rfu10/scada/operator/internal/historian/influxdb"
//	mgr.Add(historian.New(influxURL, token, org, bucket))
//	tagReconciler.Historian = historian.New(...)
//
// Until then the NoopHistorian silently drops all writes.
type HistorianSink interface {
	Record(ctx context.Context, measurement, tag string, value interface{}, quality string, ts time.Time) error
}

// NoopHistorian satisfies HistorianSink without doing anything.
type NoopHistorian struct{}

func (NoopHistorian) Record(_ context.Context, _, _ string, _ interface{}, _ string, _ time.Time) error {
	return nil
}
