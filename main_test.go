package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	cw "github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	godotenv "github.com/joho/godotenv"
	kubeappsv1 "k8s.io/api/apps/v1"
	kubemetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

var dotEnv map[string]string

// TestMain is the main entry point for all tests. It loads environment
// variables from a the env file in the project root.
func TestMain(m *testing.M) {
	var err error

	dotEnv, err = godotenv.Read()
	if err != nil {
		panic(err)
	}

	m.Run()
}

// newLogger creates a new logger that can be passed around as required.
func newLogger(t *testing.T) *slog.Logger {
	t.Helper()

	var buf bytes.Buffer

	logger := slog.New(
		slog.NewTextHandler(
			&buf,
			&slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true},
		),
	)

	t.Cleanup(func() {
		if t.Failed() || testing.Verbose() {
			fmt.Fprint(os.Stderr, buf.String())
		}
	})

	return logger
}

// TestRunMain_Version tests that the runMain function correctly handles
// the --version and --verbose flags.
func TestRunMain_Version(t *testing.T) {
	for _, tc := range []struct {
		name       string
		args       []string
		expOutputs []string
	}{{
		name:       "Version",
		args:       []string{"kubestatus2cloudwatch", "--version"},
		expOutputs: []string{"Kubestatus2cloudwatch n/a"},
	}, {
		name: "VersionVerbose",
		args: []string{"kubestatus2cloudwatch", "--verbose", "--version"},
		expOutputs: []string{
			"Program: Kubestatus2cloudwatch",
			"Version: n/a",
			"BuildDate: n/a",
			"GitCommit: n/a",
		},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			log := newLogger(t)

			originalArgs := os.Args
			originalStdout := os.Stdout

			t.Cleanup(func() {
				os.Args = originalArgs
				os.Stdout = originalStdout
			})

			os.Args = tc.args

			readPipe, writePipe, err := os.Pipe()
			if err != nil {
				t.Fatalf("Failed to create pipe: %v", err)
			}

			os.Stdout = writePipe

			runMain(t.Context(), log)

			writePipe.Close()

			var stdout bytes.Buffer

			_, err = stdout.ReadFrom(readPipe)
			if err != nil {
				t.Fatalf("Failed to read from pipe: %v", err)
			}

			t.Log(stdout.String())

			output := stdout.String()
			for _, expOutput := range tc.expOutputs {
				if !strings.Contains(output, expOutput) {
					t.Errorf("Unexpected output: %v, expected to contain: %v",
						output,
						expOutput,
					)
				}
			}
		})
	}
}

// TestRunMain_Config tests that the runMain function correctly handles
// the --config flag pointing to something different than a proper file.
func TestRunMain_Config(t *testing.T) {
	log := newLogger(t)

	originalArgs := os.Args
	originalStderr := os.Stderr

	t.Cleanup(func() {
		os.Args = originalArgs
		os.Stderr = originalStderr
	})

	os.Args = []string{"kubestatus2cloudwatch", "--config", t.TempDir()}

	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	os.Stderr = writePipe

	exitStatus := runMain(t.Context(), log)

	writePipe.Close()

	var stdout bytes.Buffer

	_, err = stdout.ReadFrom(readPipe)
	if err != nil {
		t.Fatalf("Failed to read from pipe: %v", err)
	}

	t.Log(stdout.String())

	if exitStatus != 1 {
		t.Errorf("Unexpected exit status: got %d, want %d", exitStatus, 1)
	}
}

// TestIsFittingMode tests the isFittingMode function.
func TestIsFittingMode(t *testing.T) {
	for _, tc := range []struct {
		name    string // Name of test case.
		mode    string // Fitting mode.
		got     int    // Present number.
		want    int    // Expected number.
		fitting bool   // Is it fitting the mode?
	}{{
		name:    "UnknownMode",
		mode:    "DoesNotExist",
		got:     1,
		want:    1,
		fitting: false,
	}, {
		name:    "AotFitting",
		mode:    modeAllOfThem,
		got:     3,
		want:    3,
		fitting: true,
	}, {
		name:    "AotNotFitting",
		mode:    modeAllOfThem,
		got:     3,
		want:    5,
		fitting: false,
	}, {
		name:    "AotFittingZero",
		mode:    modeAllOfThem,
		got:     0,
		want:    0,
		fitting: true,
	}, {
		name:    "AloFittingZero",
		mode:    modeAtLeastOne,
		got:     3,
		want:    0,
		fitting: true,
	}, {
		name:    "AloFitting",
		mode:    modeAtLeastOne,
		got:     1,
		want:    8,
		fitting: true,
	}, {
		name:    "AloNotFitting",
		mode:    modeAtLeastOne,
		got:     0,
		want:    3,
		fitting: false,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.fitting != isFittingMode(tc.mode, tc.got, tc.want) {
				t.Errorf("Unexpected return: got %v, want %v",
					tc.fitting,
					!tc.fitting,
				)
			}
		})
	}
}

// TestPerformScan tests the performScan function.
func TestPerformScan(t *testing.T) {
	for _, tc := range []struct {
		name             string               // Name of test case.
		objects          []kuberuntime.Object // Kubernetes objects.
		targets          []target             // Targets to scan.
		expResultSuccess []bool               // Expected result success status.
		expResultReady   []bool               // Expected result ready status.
		expScanSuccess   bool                 // Expected scan success status.
		expScanReady     bool                 // Expected scan ready status.
	}{{
		name: "DaemonSetQueryFailure",
		targets: []target{
			{kindDaemonSet, "Foo", "Baz", modeAllOfThem},
		},
		expResultSuccess: []bool{false},
		expResultReady:   []bool{false},
		expScanSuccess:   false,
		expScanReady:     false,
	}, {
		name: "DeploymentQueryFailure",
		targets: []target{
			{kindDeployment, "Foo", "Baz", modeAllOfThem},
		},
		expResultSuccess: []bool{false},
		expResultReady:   []bool{false},
		expScanSuccess:   false,
		expScanReady:     false,
	}, {
		name: "StatefulsetQueryFailure",
		targets: []target{
			{kindStatefulSet, "Foo", "Baz", modeAllOfThem},
		},
		expResultSuccess: []bool{false},
		expResultReady:   []bool{false},
		expScanSuccess:   false,
		expScanReady:     false,
	}, {
		name: "UnsupportedKind",
		targets: []target{
			{"NotSupported", "Foo", "Baz", modeAllOfThem},
		},
		expResultSuccess: []bool{false},
		expResultReady:   []bool{false},
		expScanSuccess:   false,
		expScanReady:     false,
	}, {
		name: "DaemonsetSuccessReady",
		objects: []kuberuntime.Object{
			&kubeappsv1.DaemonSet{
				ObjectMeta: kubemetav1.ObjectMeta{
					Namespace: "Foo",
					Name:      "Baz",
				},
				Status: kubeappsv1.DaemonSetStatus{
					DesiredNumberScheduled: 1,
					NumberReady:            1,
				},
			},
		},
		targets: []target{
			{kindDaemonSet, "Foo", "Baz", modeAllOfThem},
		},
		expResultSuccess: []bool{true},
		expResultReady:   []bool{true},
		expScanSuccess:   true,
		expScanReady:     true,
	}, {
		name: "DeploymentSuccessReady",
		objects: []kuberuntime.Object{
			&kubeappsv1.Deployment{
				ObjectMeta: kubemetav1.ObjectMeta{
					Namespace: "Foo",
					Name:      "Baz",
				},
				Status: kubeappsv1.DeploymentStatus{
					Replicas:      1,
					ReadyReplicas: 1,
				},
			},
		},
		targets: []target{
			{kindDeployment, "Foo", "Baz", modeAllOfThem},
		},
		expResultSuccess: []bool{true},
		expResultReady:   []bool{true},
		expScanSuccess:   true,
		expScanReady:     true,
	}, {
		name: "StatefulSetSuccessReady",
		objects: []kuberuntime.Object{
			&kubeappsv1.StatefulSet{
				ObjectMeta: kubemetav1.ObjectMeta{
					Namespace: "Foo",
					Name:      "Baz",
				},
				Status: kubeappsv1.StatefulSetStatus{
					Replicas:      1,
					ReadyReplicas: 1,
				},
			},
		},
		targets: []target{
			{kindStatefulSet, "Foo", "Baz", modeAllOfThem},
		},
		expResultSuccess: []bool{true},
		expResultReady:   []bool{true},
		expScanSuccess:   true,
		expScanReady:     true,
	}, {
		name: "StatefulsetSuccessNotReady",
		objects: []kuberuntime.Object{
			&kubeappsv1.StatefulSet{
				ObjectMeta: kubemetav1.ObjectMeta{
					Namespace: "Foo",
					Name:      "Baz",
				},
				Status: kubeappsv1.StatefulSetStatus{
					Replicas:      5,
					ReadyReplicas: 1,
				},
			},
		},
		targets: []target{
			{kindStatefulSet, "Foo", "Baz", modeAllOfThem},
		},
		expResultSuccess: []bool{true},
		expResultReady:   []bool{false},
		expScanSuccess:   true,
		expScanReady:     false,
	}, {
		name: "FailureMultiMix",
		objects: []kuberuntime.Object{
			&kubeappsv1.StatefulSet{
				ObjectMeta: kubemetav1.ObjectMeta{
					Namespace: "Foo",
					Name:      "Baz",
				},
				Status: kubeappsv1.StatefulSetStatus{
					Replicas:      5,
					ReadyReplicas: 1,
				},
			},
			&kubeappsv1.Deployment{
				ObjectMeta: kubemetav1.ObjectMeta{
					Namespace: "Foo",
					Name:      "Baz",
				},
				Status: kubeappsv1.DeploymentStatus{
					Replicas:      1,
					ReadyReplicas: 1,
				},
			},
		},
		targets: []target{
			{kindStatefulSet, "Foo", "Baz", modeAllOfThem},
			{kindDeployment, "Foo", "Baz", modeAllOfThem},
		},
		expResultSuccess: []bool{true, true},
		expResultReady:   []bool{false, true},
		expScanSuccess:   true,
		expScanReady:     false,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			log := newLogger(t)

			fakeClientset := kubefake.NewSimpleClientset(tc.objects...)
			scan := performScan(&performScanOptions{
				ctx:     t.Context(),
				log:     log,
				client:  fakeClientset,
				targets: tc.targets,
			})

			if scan.success != tc.expScanSuccess {
				t.Errorf("Unexpected scan success status: got %v, want %v",
					scan.success,
					tc.expScanSuccess,
				)
			}

			if scan.ready != tc.expScanReady {
				t.Errorf("Unexpected scan ready status: got %v, want %v",
					scan.success,
					tc.expScanSuccess,
				)
			}

			if len(scan.results) != len(tc.targets) {
				t.Errorf("Unexpected number of scan results: got %v, want %v",
					len(scan.results),
					len(tc.targets),
				)
			}

			for i := range scan.results {
				if scan.results[i].namespace != tc.targets[i].Namespace {
					t.Errorf(
						"Unexpected namespace for result %v: got %v, want %v",
						i,
						scan.results[i].namespace,
						tc.targets[i].Namespace,
					)
				}

				if scan.results[i].name != tc.targets[i].Name {
					t.Errorf(
						"Unexpected name for result %v: got %v, want %v",
						i,
						scan.results[i].name,
						tc.targets[i].Name,
					)
				}

				if scan.results[i].success != tc.expResultSuccess[i] {
					t.Errorf(
						"Unexpected success status for result %v: got %v, want %v",
						i,
						scan.results[i].success,
						tc.expResultSuccess[i],
					)
				}

				if scan.results[i].ready != tc.expResultReady[i] {
					t.Errorf(
						"Unexpected ready status for result %v: got %v, want %v",
						i,
						scan.results[i].ready,
						tc.expResultReady[i],
					)
				}
			}
		})
	}
}

// cwPutMetricDataImpl implements CWPutMetricDataAPI. Based on this example:
// https://github.com/awsdocs/aws-doc-sdk-examples/tree/main/gov2/cloudwatch/CreateCustomMetric
type cwPutMetricDataImpl struct {
	returnError bool
}

// PutMetricData implements CWPutMetricDataAPI. Based on this example:
// https://github.com/awsdocs/aws-doc-sdk-examples/tree/main/gov2/cloudwatch/CreateCustomMetric
func (dt cwPutMetricDataImpl) PutMetricData(
	_ context.Context,
	_ *cw.PutMetricDataInput,
	_ ...func(*cw.Options),
) (*cw.PutMetricDataOutput, error) {
	if dt.returnError {
		return &cw.PutMetricDataOutput{}, fmt.Errorf("fake error")
	}

	return &cw.PutMetricDataOutput{}, nil
}

// TestUpdateMetric tests the updateMetric function.
func TestUpdateMetric(t *testing.T) {
	for _, tc := range []struct {
		name        string // Name of test case.
		value       bool   // Value of metric. True is 1 and false is 0.
		dryRun      bool   // Enable dry run mode.
		returnError bool   // Should the mock return an error?
		expSuccess  bool   // Is the call expected to succeed?
	}{{
		name:        "SuccessOne",
		value:       true,
		returnError: false,
		expSuccess:  true,
	}, {
		name:        "SuccessZero",
		value:       false,
		returnError: false,
		expSuccess:  true,
	}, {
		name:        "Failure",
		value:       false,
		returnError: true,
		expSuccess:  false,
	}, {
		name:        "DryFailure",
		value:       true,
		dryRun:      true,
		returnError: true,
		expSuccess:  true,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			err := updateMetric(&updateMetricOptions{
				ctx:        t.Context(),
				dry:        tc.dryRun,
				client:     &cwPutMetricDataImpl{tc.returnError},
				namespace:  "MyNamespace",
				name:       "MyMetric",
				dimensions: []dimension{{Name: "Cluster", Value: "MyCluster"}},
				value:      tc.value,
			})

			if tc.expSuccess && err != nil {
				t.Errorf("Unexpected failure: %v", err)
			}

			if !tc.expSuccess && err == nil {
				t.Errorf("Unexpected success")
			}
		})
	}

	t.Run("SuccessNilDimensions", func(t *testing.T) {
		err := updateMetric(&updateMetricOptions{
			ctx:        t.Context(),
			dry:        false,
			client:     &cwPutMetricDataImpl{false},
			namespace:  "MyNamespace",
			name:       "MyMetric",
			dimensions: nil,
			value:      true,
		})
		if err != nil {
			t.Errorf("Unexpected failure: %v", err)
		}
	})
}

// TestExecuteRounds tests the executeRounds function.
func TestExecuteRounds(t *testing.T) {
	for _, tc := range []struct {
		name    string // Name of test case.
		cwError bool   // Should the mock return an error?
	}{{
		name:    "CloudwatchSuccess",
		cwError: false,
	}, {
		name:    "CloudwatchFailure",
		cwError: true,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			log := newLogger(t)

			err := executeRounds(&executeRoundsOptions{
				ctx:      t.Context(),
				log:      log,
				dry:      false,
				kClient:  kubefake.NewSimpleClientset(),
				cwClient: &cwPutMetricDataImpl{tc.cwError},
				single:   true,
				seconds:  1,
				metric: metric{
					Namespace:  "Namespace",
					Name:       "Name",
					Dimensions: []dimension{},
				},
				targets: []target{{
					Kind:      kindDeployment,
					Mode:      modeAllOfThem,
					Namespace: "Namespace",
					Name:      "Name",
				}},
			})

			if tc.cwError && err == nil {
				t.Errorf("Expected failure, got success")
			}

			if !tc.cwError && err != nil {
				t.Errorf("Unexpected failure: %v", err)
			}
		})
	}

	t.Run("ContextCancel", func(t *testing.T) {
		log := newLogger(t)

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		err := executeRounds(&executeRoundsOptions{
			ctx:      ctx,
			log:      log,
			dry:      false,
			kClient:  kubefake.NewSimpleClientset(),
			cwClient: &cwPutMetricDataImpl{false},
			single:   false,
			seconds:  1,
			metric: metric{
				Namespace:  "Namespace",
				Name:       "Name",
				Dimensions: []dimension{},
			},
			targets: []target{{
				Kind:      kindDeployment,
				Mode:      modeAllOfThem,
				Namespace: "Namespace",
				Name:      "Name",
			}},
		})
		if err != nil {
			t.Errorf("Unexpected failure: %v", err)
		}
	})
}
