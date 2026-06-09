package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	scadav1alpha1 "github.com/rfu10/scada/operator/api/v1alpha1"
	"github.com/rfu10/scada/operator/internal/controller"
	"github.com/rfu10/scada/operator/internal/driver"
	influxhistorian "github.com/rfu10/scada/operator/internal/historian/influxdb"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(scadav1alpha1.AddToScheme(scheme))
}

func main() {
	var (
		metricsAddr          string
		probeAddr            string
		enableLeaderElection bool
	)
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8081", "Address for the metrics endpoint.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8082", "Address for health probes.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for HA deployments.")

	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "scada.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	reg := driver.NewRegistry()

	// Wire InfluxDB historian when INFLUXDB_URL and INFLUXDB_TOKEN are set.
	// Falls back to NoopHistorian so the operator works without observability.
	var historian controller.HistorianSink = controller.NoopHistorian{}
	if influxURL := os.Getenv("INFLUXDB_URL"); influxURL != "" {
		if influxToken := os.Getenv("INFLUXDB_TOKEN"); influxToken != "" {
			org := os.Getenv("INFLUXDB_ORG")
			if org == "" {
				org = "scada"
			}
			bucket := os.Getenv("INFLUXDB_BUCKET")
			if bucket == "" {
				bucket = "historian"
			}
			h := influxhistorian.New(influxURL, influxToken, org, bucket)
			if err := mgr.Add(h); err != nil {
				setupLog.Error(err, "unable to add influxdb historian to manager")
				os.Exit(1)
			}
			historian = h
			setupLog.Info("influxdb historian enabled", "url", influxURL, "org", org, "bucket", bucket)
		}
	}

	if err = (&controller.PLCDeviceReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Registry: reg,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create PLCDevice controller")
		os.Exit(1)
	}

	if err = (&controller.TagReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Registry:  reg,
		Historian: historian,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create Tag controller")
		os.Exit(1)
	}

	if err = (&controller.SetpointReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Registry: reg,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create Setpoint controller")
		os.Exit(1)
	}

	if err = (&controller.AlarmPolicyReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("alarmpolicy-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create AlarmPolicy controller")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting scada operator")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
