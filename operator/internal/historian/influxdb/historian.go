package influxdb

import (
	"context"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	ctrl "sigs.k8s.io/controller-runtime"
)

var log = ctrl.Log.WithName("historian.influxdb")

// Historian writes tag readings to InfluxDB v2.
// It implements controller.HistorianSink and manager.Runnable so it can be
// registered with the controller-runtime manager for lifecycle management.
type Historian struct {
	url      string
	token    string
	org      string
	bucket   string
	client   influxdb2.Client
	writeAPI api.WriteAPI
}

func New(url, token, org, bucket string) *Historian {
	return &Historian{url: url, token: token, org: org, bucket: bucket}
}

// Start implements manager.Runnable. The manager calls this at startup and
// cancels the context on shutdown, giving the historian a chance to flush.
func (h *Historian) Start(ctx context.Context) error {
	h.client = influxdb2.NewClient(h.url, h.token)
	h.writeAPI = h.client.WriteAPI(h.org, h.bucket)

	// Drain async write errors so the channel never blocks.
	go func() {
		for err := range h.writeAPI.Errors() {
			log.Error(err, "async write failed")
		}
	}()

	log.Info("started", "url", h.url, "org", h.org, "bucket", h.bucket)
	<-ctx.Done()

	h.writeAPI.Flush()
	h.client.Close()
	log.Info("stopped, pending writes flushed")
	return nil
}

// Record writes a single reading to InfluxDB. Only GOOD-quality samples are
// stored; BAD/UNCERTAIN values are silently dropped to keep the historian clean.
func (h *Historian) Record(_ context.Context, measurement, tag string, value interface{}, quality string, ts time.Time) error {
	if quality != "GOOD" || h.writeAPI == nil {
		return nil
	}

	p := influxdb2.NewPoint(
		measurement,
		map[string]string{"tag_name": tag},
		map[string]interface{}{"value": value},
		ts,
	)
	h.writeAPI.WritePoint(p)
	return nil
}
