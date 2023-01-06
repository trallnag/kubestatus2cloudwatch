package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cloudwatchtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var version = ""

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	configPath, err := filepath.Abs("config.yaml")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to find config file.")
	}

	config, err := NewConfig(configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create config.")
	}

	if config.Logging.Pretty {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	if config.Logging.Level == "info" {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Info().Str("version", version).
		Msg("Welcome to KubeStatus2CloudWatch.")

	// Kubernetes setup.
	var kconfig *rest.Config
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		log.Info().Msg("Creating in-cluster Kubernetes config.")
		kconfig, err = rest.InClusterConfig()
	} else {
		log.Info().Msg("Creating out-of-cluster Kubernetes config.")
		kconfig, err = clientcmd.BuildConfigFromFlags(
			"", filepath.Join(homedir.HomeDir(), ".kube", "config"),
		)
	}
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Kubernetes config.")
	}
	log.Info().Msg("Creating Kubernetes client set.")
	kubeClient, err := kubernetes.NewForConfig(kconfig)
	if err != nil {
		log.Fatal().Err(err).
			Msg("Failed to create Kubernetes client.")
	}

	// AWS CloudWatch setup
	log.Info().Msg("Creating AWS config.")
	awsConfig, err := awsconfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal().Err(err).
			Msg("Failed to create AWS SDK config.")
	}
	cloudwatchClient := cloudwatch.NewFromConfig(awsConfig)

	log.Info().Msg("Done with setup. Starting aggregation.")

	ExecuteRounds(
		false, config.Seconds, config.Dry, config.Metric, config.Targets,
		kubeClient, cloudwatchClient,
	)
}

// ExecuteRounds indefinitely executes tick rounds (except if single is true,
// in which case only a single tick round is executed). From the config the
// keys seconds, dry, metric, and targets are used. The clients for Kubernetes
// and CloudWatch are expected to be ready to use.
func ExecuteRounds(
	single bool, seconds int, dry bool, metric Metric, targets []Target,
	kubeClient kubernetes.Interface, cloudwatchClient CWPutMetricDataAPI,
) {
	tickCount := 0
	ticker := time.NewTicker(time.Duration(seconds) * time.Second)
	for ; true; <-ticker.C {
		tickCount = tickCount + 1
		tickStart := time.Now()
		tickLogger := log.With().Int("tickCount", tickCount).Logger()
		tickLogger.Info().Msg("Executing new tick round.")

		err := UpdateMetric(
			dry, cloudwatchClient, metric.Namespace,
			metric.Name, metric.Dimensions,
			PerformScan(kubeClient, targets).Success,
		)
		if err != nil {
			tickLogger.Error().Err(err).Msg("Failed to update metric.")
		}

		tickDuration := time.Since(tickStart).Truncate(time.Millisecond)
		tickLogger.Info().
			Str("tickDuration", tickDuration.String()).
			Msg("Done with tick round.")

		if single {
			ticker.Stop()
			break
		}
	}
}

// IsFittingMode checks if the given and expected number of target instances is
// fitting the mode. Two modes are supported: "AllOfThem" requires all replicas
// to be ready. "AtLeastOne" requires at least one replica to be ready.
func IsFittingMode(mode string, got int, want int) bool {
	switch mode {
	case ModeAllOfThem:
		return got == want
	case ModeAtLeastOne:
		return want == 0 || got > 0
	default:
		return false
	}
}

// Scan stores data regarding a single scan run (that includes one or more
// targets). The "Success" field is false of one or more target scans failed
// or did not match expected condition and status.
type Scan struct {
	// Success is false if at least one target scan failed for example due to
	// the target resource not being found or a network error while calling.
	Success bool `json:"success"`

	// Shows if all targets are ready or if at least one target is not ready.
	Ready bool `json:"ready"`

	// List of all results by target. One result per target.
	Results []Result `json:"results"`
}

// Result holds information regarding a single target scan in a scan. Based on
// the given target configuration. Several fields are simply passed through.
type Result struct {
	// Was the scan for this target successful? Meaning did the Kubernetes API
	// query succeed or not (for example due to target not existing). Only if
	// this is true, "Ready" is valid.
	Success bool `json:"success"`

	// Shows if the target is ready or not according to configured mode.
	Ready bool `json:"ready"`

	// Type of the scanned target. For example "Deployment" or "StatefulSet".
	Kind string `json:"kind"`

	// Namespace of the scanned target. For example "kube-system".
	Namespace string `json:"namespace"`

	// Name of the scanned target. For example "prometheus".
	Name string `json:"name"`

	// Mode of the scanned target. For example "AllOfThem".
	Mode string `json:"mode"`

	Got  int `json:"got"`
	Want int `json:"want"`
}

// PerformScan queries the Kubernetes API for all given targets and checks
// condition status of the respective resources. The results of individual
// targets is collected in the returned struct.
//
// Currently only resources of the kind Deployment are supported.
func PerformScan(kubeClient kubernetes.Interface, targets []Target) Scan {
	scan := Scan{Success: true, Ready: true}

	for _, target := range targets {
		result := Result{
			Success:   true,
			Ready:     true,
			Kind:      target.Kind,
			Namespace: target.Namespace,
			Name:      target.Name,
			Mode:      target.Mode,
		}

		switch target.Kind {
		case KindDaemonSet:
			daemonSet, err := kubeClient.AppsV1().
				DaemonSets(target.Namespace).
				Get(context.TODO(), target.Name, metav1.GetOptions{})
			if err != nil {
				log.Error().Err(err).Msg("Failed to query Kubernetes API.")
				scan.Success, scan.Ready = false, false
				result.Success, result.Ready = false, false
			} else {
				result.Got = int(daemonSet.Status.DesiredNumberScheduled)
				result.Want = int(daemonSet.Status.NumberReady)
			}
		case KindDeployment:
			deployment, err := kubeClient.AppsV1().
				Deployments(target.Namespace).
				Get(context.TODO(), target.Name, metav1.GetOptions{})
			if err != nil {
				log.Error().Err(err).Msg("Failed to query Kubernetes API.")
				scan.Success, scan.Ready = false, false
				result.Success, result.Ready = false, false
			} else {
				result.Got = int(deployment.Status.Replicas)
				result.Want = int(deployment.Status.ReadyReplicas)
			}
		case KindStatefulSet:
			statefulSet, err := kubeClient.AppsV1().
				StatefulSets(target.Namespace).
				Get(context.TODO(), target.Name, metav1.GetOptions{})
			if err != nil {
				log.Error().Err(err).Msg("Failed to query Kubernetes API.")
				scan.Success, scan.Ready = false, false
				result.Success, result.Ready = false, false
			} else {
				result.Got = int(statefulSet.Status.Replicas)
				result.Want = int(statefulSet.Status.ReadyReplicas)
			}
		default:
			scan.Success, scan.Ready = false, false
			result.Success, result.Ready = false, false
		}

		if result.Success {
			result.Ready = IsFittingMode(target.Mode, result.Got, result.Want)
		}

		if !result.Ready {
			scan.Ready = false
		}

		scan.Results = append(scan.Results, result)
	}

	scanJSON, err := json.Marshal(scan)
	if err != nil {
		log.Error().Err(err).Msg("Extraordinary error.")
	}
	if scan.Success && scan.Ready {
		log.Debug().RawJSON("scan", scanJSON).
			Msg("Done with scan. All looking good.")
	} else {
		log.Info().RawJSON("scan", scanJSON).
			Msg("Done with scan. Something is wrong.")
	}

	return scan
}

// CWPutMetricDataAPI defines the interface for the PutMetricData function.
// We use this interface to test the function using a mocked service.
type CWPutMetricDataAPI interface {
	PutMetricData(ctx context.Context,
		params *cloudwatch.PutMetricDataInput,
		optFns ...func(*cloudwatch.Options),
	) (*cloudwatch.PutMetricDataOutput, error)
}

// UpdateMetric updates a metric using PutMetricData. The value of the metric
// is binary. That's why "value" is a boolean. The parameters "namespace" and
// "name" must be set, while "dimensions" can be an empty list.
func UpdateMetric(
	dry bool,
	client CWPutMetricDataAPI,
	namespace string,
	name string,
	dimensions []Dimension,
	value bool,
) error {
	metricValue := 0.0
	if value {
		metricValue = 1.0
	}

	metricDimensions := []cloudwatchtypes.Dimension{}
	for _, configDimension := range dimensions {
		metricDimensions = append(metricDimensions, cloudwatchtypes.Dimension{
			Name:  aws.String(configDimension.Name),
			Value: aws.String(configDimension.Value),
		})
	}

	input := &cloudwatch.PutMetricDataInput{
		Namespace: aws.String(namespace),
		MetricData: []cloudwatchtypes.MetricDatum{{
			MetricName: aws.String(name),
			Unit:       cloudwatchtypes.StandardUnitNone,
			Value:      aws.Float64(metricValue),
			Dimensions: metricDimensions,
		}},
	}

	if !dry {
		_, err := client.PutMetricData(context.TODO(), input)
		if err != nil {
			return fmt.Errorf("failed to update metric: %w", err)
		}
	}

	return nil
}
