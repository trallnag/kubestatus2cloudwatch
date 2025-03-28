package main

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	cw "github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	sts "github.com/aws/aws-sdk-go-v2/service/sts"
	kubemetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kube "k8s.io/client-go/kubernetes"
	kuberest "k8s.io/client-go/rest"
	kubeclientcmd "k8s.io/client-go/tools/clientcmd"
)

var (
	program   = "Kubestatus2cloudwatch"
	version   = "n/a"
	buildDate = "n/a"
	gitCommit = "n/a"
)

func main() {
	ctx := context.Background()

	exitCode := runMain(ctx, nil)
	os.Exit(exitCode)
}

// runMain is the main entry point for the program. It parses command line
// arguments, creates a logger if the passed logger is nil, and executes the
// main logic of the program. The return value represents the exit status.
func runMain(ctx context.Context, log *slog.Logger) int {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	configFlag := flag.String(
		"config",
		"",
		"Path to the configuration file.",
	)
	verboseFlag := flag.Bool(
		"verbose",
		false,
		"Make the program more talkative.",
	)
	versionFlag := flag.Bool(
		"version",
		false,
		"Print version information and exit.",
	)

	flag.Parse()

	if *versionFlag {
		if *verboseFlag {
			fmt.Fprintf(
				os.Stdout,
				"Program: %s\n"+
					"Version: %s\n"+
					"BuildDate: %s\n"+
					"GitCommit: %s\n",
				program, version, buildDate, gitCommit,
			)
		} else {
			fmt.Fprintf(os.Stdout, "%s %s\n", program, version)
		}

		return 0
	}

	config, err := newConfig(cmp.Or(
		*configFlag, os.Getenv("KS2CW_CONFIG_PATH"), "config.yaml",
	))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create config: %v\n", err)

		return 1
	}

	if log == nil {
		handlerOptions := &slog.HandlerOptions{
			AddSource:   false,
			Level:       slog.LevelInfo,
			ReplaceAttr: nil,
		}

		if *verboseFlag {
			handlerOptions.AddSource = true
		}

		if config.Logging.Level == logLevelDebug {
			handlerOptions.Level = slog.LevelDebug
		}

		if config.Logging.Format == logFormatJSON {
			log = slog.New(slog.NewJSONHandler(os.Stderr, handlerOptions))
		} else {
			log = slog.New(slog.NewTextHandler(os.Stderr, handlerOptions))
		}
	}

	log.Info("Program information",
		slog.String("program", program),
		slog.String("version", version),
		slog.String("buildDate", buildDate),
		slog.String("gitCommit", gitCommit),
	)

	kubernetesClient, err := newKubernetesClient()
	if err != nil {
		log.Error("Failed to create Kubernetes client.",
			slog.Any("error", err),
		)

		return 1
	}

	cloudwatchClient, err := newCloudwatchClient(ctx)
	if err != nil {
		log.Error("Failed to create CloudWatch client.",
			slog.Any("error", err),
		)

		return 1
	}

	if err = executeRounds(&executeRoundsOptions{
		ctx:      ctx,
		log:      log,
		dry:      config.DryRun,
		kClient:  kubernetesClient,
		cwClient: cloudwatchClient,
		single:   false,
		seconds:  config.Seconds,
		metric:   config.Metric,
		targets:  config.Targets,
	}); err != nil {
		log.Error("Failure during round execution.",
			slog.Any("error", err),
		)

		return 1
	}

	return 0
}

// newKubernetesClient creates and configures a new Kubernetes client.
func newKubernetesClient() (*kube.Clientset, error) {
	var config *kuberest.Config

	var err error

	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		config, err = kuberest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf(
				"create in-cluster Kubernetes config: %v", err,
			)
		}
	} else {
		config, err = kubeclientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			kubeclientcmd.NewDefaultClientConfigLoadingRules(),
			&kubeclientcmd.ConfigOverrides{},
		).ClientConfig()
		if err != nil {
			return nil, fmt.Errorf(
				"create out-of-cluster Kubernetes config: %v", err,
			)
		}
	}

	client, err := kube.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf(
			"create Kubernetes client: %v", err,
		)
	}

	if _, err = client.Discovery().ServerVersion(); err != nil {
		return nil, fmt.Errorf(
			"get Kubernetes server version: %v", err,
		)
	}

	return client, nil
}

// newCloudwatchClient creates and configures a new CloudWatch client.
func newCloudwatchClient(ctx context.Context) (*cw.Client, error) {
	config, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("create AWS SDK config: %v", err)
	}

	stsClient := sts.NewFromConfig(config)

	if _, err = stsClient.GetCallerIdentity(
		ctx, &sts.GetCallerIdentityInput{},
	); err != nil {
		return nil, fmt.Errorf("get AWS caller identity: %v", err)
	}

	return cw.NewFromConfig(config), nil
}

// executeRoundsOptions holds the input for the executeRounds function.
type executeRoundsOptions struct {
	ctx context.Context
	log *slog.Logger
	dry bool

	// Clients for Kubernetes and CloudWatch.
	kClient  kube.Interface
	cwClient cwPutMetricDataAPI

	// Single run flag. If enabled, only a single tick round is executed.
	single bool

	// Seconds between tick rounds.
	seconds int

	// Metric to update.
	metric metric

	// Targets to scan.
	targets []target
}

// executeRounds executes tick rounds.
func executeRounds(o *executeRoundsOptions) error {
	tickCount := 0

	ticker := time.NewTicker(time.Duration(o.seconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-o.ctx.Done():
			o.log.Info("Received shutdown signal. Stopping.")

			return nil
		case <-ticker.C:
			tickCount++
			tickStart := time.Now()
			tickLog := o.log.With(slog.Int("tickCount", tickCount))
			tickLog.Info("Executing new tick round.")

			scan := performScan(&performScanOptions{
				ctx:     o.ctx,
				log:     tickLog,
				client:  o.kClient,
				targets: o.targets,
			})

			if err := updateMetric(&updateMetricOptions{
				ctx:        o.ctx,
				dry:        o.dry,
				client:     o.cwClient,
				namespace:  o.metric.Namespace,
				name:       o.metric.Name,
				dimensions: o.metric.Dimensions,
				value:      scan.ready,
			}); err != nil {
				return fmt.Errorf("update metric: %v", err)
			}

			tickDuration := time.Since(tickStart).Truncate(time.Millisecond)
			tickLog.Info("Done with tick round",
				slog.String("duration", tickDuration.String()),
			)

			if o.single {
				tickLog.Info("Single round requested. Stopping.")

				return nil
			}
		}
	}
}

// isFittingMode checks if the given and expected number of target instances is
// fitting the mode. Two modes are supported: "AllOfThem" requires all replicas
// to be ready. "AtLeastOne" requires at least one replica to be ready.
func isFittingMode(mode string, got int, want int) bool {
	switch mode {
	case modeAllOfThem:
		return got == want
	case modeAtLeastOne:
		return want == 0 || got > 0
	default:
		return false
	}
}

// scan stores data regarding a single scan run (that includes one or more
// targets). The "success" field is false of one or more target scans failed
// or did not match expected condition and status.
type scan struct {
	// success is false if at least one target scan failed for example due to
	// the target resource not being found or a network error while calling.
	success bool

	// Shows if all targets are ready or if at least one target is not ready.
	// Basically, if at least one result in the list of results is success=false
	// or ready=false, this is false.
	ready bool

	// List of all results by target. One result per target.
	results []result
}

// result holds information regarding a single target scan in a scan. Based on
// the given target configuration. Several fields are simply passed through.
type result struct {
	// Was the scan for this target successful? Meaning did the Kubernetes API
	// query succeed or not (for example due to target not existing). Only if
	// this is true, "ready" is valid.
	success bool

	// Shows if the target is ready or not according to configured mode.
	ready bool

	// Type of the scanned target. For example "Deployment" or "StatefulSet".
	kind string

	// namespace of the scanned target. For example "kube-system".
	namespace string

	// name of the scanned target. For example "prometheus".
	name string

	// mode of the scanned target. For example "AllOfThem".
	mode string

	// Ready information.
	got  int
	want int
}

// performScanOptions holds the input for the performScan function.
type performScanOptions struct {
	ctx context.Context
	log *slog.Logger

	// Kubernetes client.
	client kube.Interface

	// Targets to scan.
	targets []target
}

// performScan queries the Kubernetes API for all given targets and checks
// condition status of the respective resources. The results of individual
// target scans are collected in the returned struct. Errors are logged and then
// swallowed.
func performScan(o *performScanOptions) scan {
	scan := scan{success: true, ready: true, results: nil}

	for _, target := range o.targets {
		result := result{
			success:   true,
			ready:     true,
			kind:      target.Kind,
			namespace: target.Namespace,
			name:      target.Name,
			mode:      target.Mode,
			got:       0,
			want:      0,
		}

		switch target.Kind {
		case kindDaemonSet:
			daemonSet, err := o.client.AppsV1().
				DaemonSets(target.Namespace).
				Get(o.ctx, target.Name, kubemetav1.GetOptions{})
			if err != nil {
				o.log.Error("Failed to query Kubernetes API.",
					slog.Any("error", err),
				)

				scan.success, scan.ready = false, false
				result.success, result.ready = false, false
			} else {
				result.got = int(daemonSet.Status.DesiredNumberScheduled)
				result.want = int(daemonSet.Status.NumberReady)
			}
		case kindDeployment:
			deployment, err := o.client.AppsV1().
				Deployments(target.Namespace).
				Get(o.ctx, target.Name, kubemetav1.GetOptions{})
			if err != nil {
				o.log.Error("Failed to query Kubernetes API.",
					slog.Any("error", err),
				)

				scan.success, scan.ready = false, false
				result.success, result.ready = false, false
			} else {
				result.got = int(deployment.Status.Replicas)
				result.want = int(deployment.Status.ReadyReplicas)
			}
		case kindStatefulSet:
			statefulSet, err := o.client.AppsV1().
				StatefulSets(target.Namespace).
				Get(o.ctx, target.Name, kubemetav1.GetOptions{})
			if err != nil {
				o.log.Error("Failed to query Kubernetes API.",
					slog.Any("error", err),
				)

				scan.success, scan.ready = false, false
				result.success, result.ready = false, false
			} else {
				result.got = int(statefulSet.Status.Replicas)
				result.want = int(statefulSet.Status.ReadyReplicas)
			}
		default:
			scan.success, scan.ready = false, false
			result.success, result.ready = false, false
		}

		if result.success {
			result.ready = isFittingMode(target.Mode, result.got, result.want)
		}

		if !result.ready {
			scan.ready = false
		}

		scan.results = append(scan.results, result)
	}

	if scan.success && scan.ready {
		o.log.Debug("Done with scan. All looking good.", slog.Any("scan", scan))
	} else {
		o.log.Warn("Done with scan. Something is wrong.", slog.Any("scan", scan))
	}

	return scan
}

// cwPutMetricDataAPI defines the interface for the PutMetricData function.
// We use this interface to test the function using a mocked service.
type cwPutMetricDataAPI interface {
	PutMetricData(ctx context.Context,
		params *cw.PutMetricDataInput,
		optFns ...func(*cw.Options),
	) (*cw.PutMetricDataOutput, error)
}

// updateMetricOptions holds the input for the updateMetric function.
type updateMetricOptions struct {
	ctx context.Context
	dry bool

	// CloudWatch client with required interface.
	client cwPutMetricDataAPI

	// Namespace and name of the CloudWatch metric to update.
	namespace string
	name      string

	// Dimensions of the CloudWatch metric to update.
	dimensions []dimension

	// Value of the CloudWatch metric to update.
	value bool
}

// updateMetric updates a CloudWatch metric using PutMetricData.
func updateMetric(o *updateMetricOptions) error {
	if o.dimensions == nil {
		o.dimensions = []dimension{}
	}

	metricValue := 0.0
	if o.value {
		metricValue = 1.0
	}

	metricDimensions := []cwtypes.Dimension{}
	for _, configDimension := range o.dimensions {
		metricDimensions = append(metricDimensions, cwtypes.Dimension{
			Name:  aws.String(configDimension.Name),
			Value: aws.String(configDimension.Value),
		})
	}

	if !o.dry {
		_, err := o.client.PutMetricData(o.ctx,
			&cw.PutMetricDataInput{
				Namespace: aws.String(o.namespace),
				MetricData: []cwtypes.MetricDatum{{
					MetricName: aws.String(o.name),
					Unit:       cwtypes.StandardUnitNone,
					Value:      aws.Float64(metricValue),
					Dimensions: metricDimensions,
				}},
			},
		)
		if err != nil {
			return fmt.Errorf("update metric: %v", err)
		}
	}

	return nil
}
