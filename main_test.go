package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestMain(m *testing.M) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	m.Run()
}

// TestMain_Version tests that Main prints version info.
func TestMain_Version(t *testing.T) {
	originalArgs := os.Args
	originalStdout := os.Stdout

	defer func() {
		os.Args = originalArgs
		os.Stdout = originalStdout
	}()

	os.Args = []string{"kubestatus2cloudwatch", "--version"}

	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	os.Stdout = writePipe

	RunMain()
	writePipe.Close()

	var stdout bytes.Buffer

	_, err = stdout.ReadFrom(readPipe)
	if err != nil {
		t.Fatalf("Failed to read from pipe: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Kubestatus2cloudwatch n/a") {
		t.Errorf("unexpected output: %s", output)
	}
}

// TestMain_VersionVerbose tests that Main prints verbose version info.
func TestMain_VersionVerbose(t *testing.T) {
	originalArgs := os.Args
	originalStdout := os.Stdout

	defer func() {
		os.Args = originalArgs
		os.Stdout = originalStdout
	}()

	os.Args = []string{"kubestatus2cloudwatch", "--verbose", "--version"}

	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	os.Stdout = writePipe

	RunMain()
	writePipe.Close()

	var stdout bytes.Buffer

	_, err = stdout.ReadFrom(readPipe)
	if err != nil {
		t.Fatalf("Failed to read from pipe: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Program: Kubestatus2cloudwatch") ||
		!strings.Contains(output, "Version: n/a") ||
		!strings.Contains(output, "BuildDate: n/a") ||
		!strings.Contains(output, "GitCommit: n/a") {
		t.Errorf("unexpected output: %s", output)
	}
}

// TestIsFittingMode tests IsFittingMode.
func TestIsFittingMode(t *testing.T) {
	for _, tc := range []struct {
		name    string // Name of test case.
		mode    string // Fitting mode.
		got     int    // Present number.
		want    int    // Expected number.
		fitting bool   // Is it fitting the mode?
	}{{
		name:    "1_unknown_mode",
		mode:    "DoesNotExist",
		got:     1,
		want:    1,
		fitting: false,
	}, {
		name:    "2_aot_fitting",
		mode:    ModeAllOfThem,
		got:     3,
		want:    3,
		fitting: true,
	}, {
		name:    "3_aot_not_fitting",
		mode:    ModeAllOfThem,
		got:     3,
		want:    5,
		fitting: false,
	}, {
		name:    "4_aot_fitting_zero",
		mode:    ModeAllOfThem,
		got:     0,
		want:    0,
		fitting: true,
	}, {
		name:    "5_alo_fitting_zero",
		mode:    ModeAtLeastOne,
		got:     3,
		want:    0,
		fitting: true,
	}, {
		name:    "5_alo_fitting",
		mode:    ModeAtLeastOne,
		got:     1,
		want:    8,
		fitting: true,
	}, {
		name:    "5_alo_not_fitting",
		mode:    ModeAtLeastOne,
		got:     0,
		want:    3,
		fitting: false,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.fitting != IsFittingMode(tc.mode, tc.got, tc.want) {
				t.Errorf(
					"Unexpected return: got %v, want %v",
					tc.fitting, !tc.fitting,
				)
			}
		})
	}
}

// TestPerformScan tests PerformScan.
func TestPerformScan(t *testing.T) {
	for _, tc := range []struct {
		name             string           // Name of test case.
		objects          []runtime.Object // Kubernetes objects.
		targets          []Target         // Targets to scan.
		expResultSuccess []bool           // Expected result success status.
		expResultReady   []bool           // Expected result ready status.
		expScanSuccess   bool             // Expected scan success status.
		expScanReady     bool             // Expected scan ready status.
	}{{
		name: "1_daemonset_query_failure",
		targets: []Target{
			{KindDaemonSet, "Foo", "Baz", ModeAllOfThem},
		},
		expResultSuccess: []bool{false},
		expResultReady:   []bool{false},
		expScanSuccess:   false,
		expScanReady:     false,
	}, {
		name: "2_deployment_query_failure",
		targets: []Target{
			{KindDeployment, "Foo", "Baz", ModeAllOfThem},
		},
		expResultSuccess: []bool{false},
		expResultReady:   []bool{false},
		expScanSuccess:   false,
		expScanReady:     false,
	}, {
		name: "3_statefulset_query_failure",
		targets: []Target{
			{KindStatefulSet, "Foo", "Baz", ModeAllOfThem},
		},
		expResultSuccess: []bool{false},
		expResultReady:   []bool{false},
		expScanSuccess:   false,
		expScanReady:     false,
	}, {
		name: "4_unsupported_kind",
		targets: []Target{
			{"NotSupported", "Foo", "Baz", ModeAllOfThem},
		},
		expResultSuccess: []bool{false},
		expResultReady:   []bool{false},
		expScanSuccess:   false,
		expScanReady:     false,
	}, {
		name: "5_daemonset_success_ready",
		objects: []runtime.Object{
			&appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "Foo",
					Name:      "Baz",
				},
				Status: appsv1.DaemonSetStatus{
					DesiredNumberScheduled: 1,
					NumberReady:            1,
				},
			},
		},
		targets: []Target{
			{KindDaemonSet, "Foo", "Baz", ModeAllOfThem},
		},
		expResultSuccess: []bool{true},
		expResultReady:   []bool{true},
		expScanSuccess:   true,
		expScanReady:     true,
	}, {
		name: "6_deployment_success_ready",
		objects: []runtime.Object{
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "Foo",
					Name:      "Baz",
				},
				Status: appsv1.DeploymentStatus{
					Replicas:      1,
					ReadyReplicas: 1,
				},
			},
		},
		targets: []Target{
			{KindDeployment, "Foo", "Baz", ModeAllOfThem},
		},
		expResultSuccess: []bool{true},
		expResultReady:   []bool{true},
		expScanSuccess:   true,
		expScanReady:     true,
	}, {
		name: "7_statefulset_success_ready",
		objects: []runtime.Object{
			&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "Foo",
					Name:      "Baz",
				},
				Status: appsv1.StatefulSetStatus{
					Replicas:      1,
					ReadyReplicas: 1,
				},
			},
		},
		targets: []Target{
			{KindStatefulSet, "Foo", "Baz", ModeAllOfThem},
		},
		expResultSuccess: []bool{true},
		expResultReady:   []bool{true},
		expScanSuccess:   true,
		expScanReady:     true,
	}, {
		name: "8_statefulset_success_not_ready",
		objects: []runtime.Object{
			&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "Foo",
					Name:      "Baz",
				},
				Status: appsv1.StatefulSetStatus{
					Replicas:      5,
					ReadyReplicas: 1,
				},
			},
		},
		targets: []Target{
			{KindStatefulSet, "Foo", "Baz", ModeAllOfThem},
		},
		expResultSuccess: []bool{true},
		expResultReady:   []bool{false},
		expScanSuccess:   true,
		expScanReady:     false,
	}, {
		name: "9_failure_multi_mix",
		objects: []runtime.Object{
			&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "Foo",
					Name:      "Baz",
				},
				Status: appsv1.StatefulSetStatus{
					Replicas:      5,
					ReadyReplicas: 1,
				},
			},
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "Foo",
					Name:      "Baz",
				},
				Status: appsv1.DeploymentStatus{
					Replicas:      1,
					ReadyReplicas: 1,
				},
			},
		},
		targets: []Target{
			{KindStatefulSet, "Foo", "Baz", ModeAllOfThem},
			{KindDeployment, "Foo", "Baz", ModeAllOfThem},
		},
		expResultSuccess: []bool{true, true},
		expResultReady:   []bool{false, true},
		expScanSuccess:   true,
		expScanReady:     false,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			fakeClientset := fake.NewSimpleClientset(tc.objects...)
			scan := PerformScan(fakeClientset, tc.targets)

			if scan.Success != tc.expScanSuccess {
				t.Fatalf(
					"Unexpected scan success status: got %v, want %v",
					scan.Success, tc.expScanSuccess,
				)
			}

			if scan.Ready != tc.expScanReady {
				t.Fatalf(
					"Unexpected scan ready status: got %v, want %v",
					scan.Success, tc.expScanSuccess,
				)
			}

			if len(scan.Results) != len(tc.targets) {
				t.Fatalf(
					"Unexpected number of scan results: got %v, want %v",
					len(scan.Results), len(tc.targets),
				)
			}

			for i := range scan.Results {
				if scan.Results[i].Namespace != tc.targets[i].Namespace {
					t.Errorf(
						"Unexpected namespace for result %v: got %v, want %v",
						i, scan.Results[i].Namespace, tc.targets[i].Namespace,
					)
				}

				if scan.Results[i].Name != tc.targets[i].Name {
					t.Errorf(
						"Unexpected name for result %v: got %v, want %v",
						i, scan.Results[i].Name, tc.targets[i].Name,
					)
				}

				if scan.Results[i].Success != tc.expResultSuccess[i] {
					t.Errorf(
						"Unexpected success status for result %v: got %v, want %v",
						i, scan.Results[i].Success, tc.expResultSuccess[i],
					)
				}

				if scan.Results[i].Ready != tc.expResultReady[i] {
					t.Errorf(
						"Unexpected ready status for result %v: got %v, want %v",
						i, scan.Results[i].Ready, tc.expResultReady[i],
					)
				}
			}
		})
	}
}

// CWPutMetricDataImpl implements CWPutMetricDataAPI. Based on the example
// provided by AWS [here] on GitHub.
//
// [here]: https://github.com/awsdocs/aws-doc-sdk-examples/tree/main/gov2/cloudwatch/CreateCustomMetric
type CWPutMetricDataImpl struct {
	returnError bool
}

// PutMetricData implements CWPutMetricDataAPI. Based on the example provided
// by AWS [here] on GitHub.
//
// [here]: https://github.com/awsdocs/aws-doc-sdk-examples/tree/main/gov2/cloudwatch/CreateCustomMetric
func (dt CWPutMetricDataImpl) PutMetricData(
	_ context.Context,
	_ *cloudwatch.PutMetricDataInput,
	_ ...func(*cloudwatch.Options),
) (*cloudwatch.PutMetricDataOutput, error) {
	if dt.returnError {
		return &cloudwatch.PutMetricDataOutput{}, fmt.Errorf("fake error")
	}

	return &cloudwatch.PutMetricDataOutput{}, nil
}

// TestUpdateMetric tests UpdateMetric.
func TestUpdateMetric(t *testing.T) {
	for _, tc := range []struct {
		name        string // Name of test case.
		value       bool   // Value of Metric. True is 1 and false is 0.
		dry         bool   // Pass dry true to updateMetric function.
		returnError bool   // Should the mock return an error?
		expSuccess  bool   // Is the call expected to succeed?
	}{{
		name:        "1_succes_one",
		value:       true,
		returnError: false,
		expSuccess:  true,
	}, {
		name:        "2_succes_zero",
		value:       false,
		returnError: false,
		expSuccess:  true,
	}, {
		name:        "3_failure",
		value:       false,
		returnError: true,
		expSuccess:  false,
	}, {
		name:        "4_dry_failure",
		value:       true,
		dry:         true,
		returnError: true,
		expSuccess:  true,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &CWPutMetricDataImpl{tc.returnError}
			err := UpdateMetric(
				tc.dry, mockClient, "MyNamespace", "MyMetric",
				[]Dimension{{Name: "Cluster", Value: "MyCluster"}},
				tc.value,
			)

			if tc.expSuccess && err != nil {
				t.Fatalf("Unexpected failure: %s", err.Error())
			}

			if !tc.expSuccess && err == nil {
				t.Fatal("Unexpected success")
			}
		})
	}
}

// TestExecuteRounds tests ExecuteRounds.
func TestExecuteRounds(_ *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	cloudwatchClient := &CWPutMetricDataImpl{false}
	ExecuteRounds(
		true,
		1,
		false,
		Metric{
			Namespace: "Namespace",
			Name:      "Name",
		},
		[]Target{{
			Kind:      KindDeployment,
			Mode:      ModeAllOfThem,
			Namespace: "Namespace",
			Name:      "Name",
		}},
		kubeClient,
		cloudwatchClient,
	)

	kubeClient = fake.NewSimpleClientset()
	cloudwatchClient = &CWPutMetricDataImpl{true}
	ExecuteRounds(
		true,
		1,
		false,
		Metric{
			Namespace: "Namespace",
			Name:      "Name",
		},
		[]Target{{
			Kind:      KindDeployment,
			Mode:      ModeAllOfThem,
			Namespace: "Namespace",
			Name:      "Name",
		}},
		kubeClient,
		cloudwatchClient,
	)
}
