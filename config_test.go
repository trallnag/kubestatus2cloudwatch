package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cmp "github.com/google/go-cmp/cmp"
	dedent "github.com/lithammer/dedent"
)

// newExampleConfig creates a valid example config.
func newExampleConfig(t *testing.T) config {
	t.Helper()

	return config{
		DryRun:  true,
		Seconds: 63,
		Logging: logging{
			Level:  "debug",
			Format: "logfmt",
		},
		Metric: metric{
			Namespace: "MyNamespace",
			Name:      "MyMetric",
			Dimensions: []dimension{
				{Name: "Cluster", Value: "MyCluster"},
			},
		},
		Targets: []target{
			{
				Kind:      kindStatefulSet,
				Namespace: "observability",
				Name:      "prometheus",
				Mode:      modeAllOfThem,
			},
		},
	}
}

// TestNewConfig tests that the function newConfig correctly parses
// configuration files. Validation of the config is not focus of this test.
func TestNewConfig(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("ErrorReadConfig", func(t *testing.T) {
		_, err := newConfig(tempDir)

		if err == nil {
			t.Fatalf("Expected error, got nil")
		}

		want := "read config"

		if !strings.Contains(err.Error(), want) {
			t.Errorf("Expected error to contain %q, got %q", want, err)
		}
	})

	t.Run("ErrorUnmarshalConfig", func(t *testing.T) {
		fileContent := "this is definitely not yaml"
		configPath := filepath.Join(tempDir, "ErrorUnmarshalConfig.yaml")

		err := os.WriteFile(configPath, []byte(fileContent), 0o600)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		_, err = newConfig(configPath)

		if err == nil {
			t.Fatalf("Expected error, got nil")
		}

		want := "unmarshal config"

		if !strings.Contains(err.Error(), want) {
			t.Errorf("Expected error to contain %q, got %q", want, err)
		}
	})

	t.Run("ErrorProcessConfig", func(t *testing.T) {
		fileContent := "this: is valid yaml but invalid config"
		configPath := filepath.Join(tempDir, "ErrorProcessConfig.yaml")

		err := os.WriteFile(configPath, []byte(fileContent), 0o600)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		_, err = newConfig(configPath)

		if err == nil {
			t.Fatalf("Expected error, got nil")
		}

		want := "process config"
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Expected error to contain %q, got %q", want, err)
		}
	})

	t.Run("Success", func(t *testing.T) {
		fileContent := dedent.Dedent(`
			dryRun: true
			seconds: 63
			logging:
			  level: debug
			  format: logfmt
			metric:
			  namespace: MyNamespace
			  name: MyMetric
			  dimensions:
			    - name: Cluster
			      value: MyCluster
			targets:
			  - kind: StatefulSet
			    namespace: observability
			    name: prometheus
			    mode: AllOfThem
		`)
		configPath := filepath.Join(tempDir, "Success.yaml")

		err := os.WriteFile(configPath, []byte(fileContent), 0o600)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		gotConfig, err := newConfig(configPath)
		if err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}

		wantConfig := config{
			DryRun:  true,
			Seconds: 63,
			Logging: logging{
				Level:  "debug",
				Format: "logfmt",
			},
			Metric: metric{
				Namespace: "MyNamespace",
				Name:      "MyMetric",
				Dimensions: []dimension{
					{Name: "Cluster", Value: "MyCluster"},
				},
			},
			Targets: []target{
				{
					Kind:      kindStatefulSet,
					Namespace: "observability",
					Name:      "prometheus",
					Mode:      modeAllOfThem,
				},
			},
		}

		if diff := cmp.Diff(wantConfig, gotConfig); diff != "" {
			t.Errorf("Config mismatch (-want +got):\n%v", diff)
		}
	})
}

// TestProcessConfig tests that the processConfig function correctly processes
// configuration instances, including validation of the metric config and targets config.
func TestProcessConfig(t *testing.T) {
	t.Run("SecondsTooSmall", func(t *testing.T) {
		config := newExampleConfig(t)
		config.Seconds = minSeconds - 1

		processedConfig, err := processConfig(config)
		if err != nil {
			t.Fatalf("Failed to process config: %v", err)
		}

		if processedConfig.Seconds != defaultSeconds {
			t.Errorf(
				"Unexpected seconds value: got %v, want %v",
				processedConfig.Seconds,
				defaultSeconds,
			)
		}
	})

	t.Run("InvalidMetric", func(t *testing.T) {
		config := newExampleConfig(t)
		config.Metric.Namespace = ""

		_, err := processConfig(config)
		if err == nil {
			t.Fatalf("Expected error, got nil")
		}

		want := "missing: metric.namespace"
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Expected error to contain %q, got %q", want, err)
		}
	})

	t.Run("InvalidTargets", func(t *testing.T) {
		config := newExampleConfig(t)
		config.Targets[0].Kind = ""

		_, err := processConfig(config)
		if err == nil {
			t.Fatalf("Expected error, got nil")
		}

		want := "missing: target[0].kind"
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Expected error to contain %q, got %q", want, err)
		}
	})

	t.Run("DefaultLogLevel", func(t *testing.T) {
		config := newExampleConfig(t)
		config.Logging.Level = ""

		processedConfig, err := processConfig(config)
		if err != nil {
			t.Fatalf("Failed to process config: %v", err)
		}

		if processedConfig.Logging.Level != logLevelInfo {
			t.Errorf(
				"Unexpected log level: got %v, want %v",
				processedConfig.Logging.Level,
				logLevelInfo,
			)
		}
	})

	t.Run("InvalidLogLevel", func(t *testing.T) {
		config := newExampleConfig(t)
		config.Logging.Level = "invalid"

		_, err := processConfig(config)
		if err == nil {
			t.Fatalf("Expected error, got nil")
		}

		want := "logging.level invalid: invalid"
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Expected error to contain %q, got %q", want, err)
		}
	})

	t.Run("DefaultLogFormat", func(t *testing.T) {
		config := newExampleConfig(t)
		config.Logging.Format = ""

		processedConfig, err := processConfig(config)
		if err != nil {
			t.Fatalf("Failed to process config: %v", err)
		}

		if processedConfig.Logging.Format != logFormatJSON {
			t.Errorf(
				"Unexpected log format: got %v, want %v",
				processedConfig.Logging.Format,
				logFormatJSON,
			)
		}
	})

	t.Run("InvalidLogFormat", func(t *testing.T) {
		config := newExampleConfig(t)
		config.Logging.Format = "invalid"

		_, err := processConfig(config)
		if err == nil {
			t.Fatalf("Expected error, got nil")
		}

		want := "logging.format invalid: invalid"
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Expected error to contain %q, got %q", want, err)
		}
	})
}

// TestValidateMetric tests the validateMetric function.
func TestValidateMetric(t *testing.T) {
	for _, tc := range []struct {
		name      string // Name of test case.
		metric    metric // Initialized metric struct.
		errSubstr string // Substring expected to be in error string.
	}{{
		name: "NamespaceEmpty",
		metric: metric{
			Name: "Name",
		},
		errSubstr: "missing: metric.namespace",
	}, {
		name: "NameEmpty",
		metric: metric{
			Namespace: "Namespace",
		},
		errSubstr: "missing: metric.name",
	}, {
		name: "DimensionNameEmpty",
		metric: metric{
			Name:       "Name",
			Namespace:  "Namespace",
			Dimensions: []dimension{{Value: "Value"}},
		},
		errSubstr: "missing: metric.dimensions[0].name",
	}, {
		name: "DimensionValueEmpty",
		metric: metric{
			Name:       "Name",
			Namespace:  "Namespace",
			Dimensions: []dimension{{Name: "Name"}},
		},
		errSubstr: "missing: metric.dimensions[0].value",
	}, {
		name: "AllIsGood",
		metric: metric{
			Name:      "Name",
			Namespace: "Namespace",
			Dimensions: []dimension{
				{Name: "Name1", Value: "Value1"},
				{Name: "Name2", Value: "Value2"},
			},
		},
	}, {
		name: "DimensionsNil",
		metric: metric{
			Name:       "Name",
			Namespace:  "Namespace",
			Dimensions: nil,
		},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateMetric(tc.metric)
			if err != nil {
				if len(tc.errSubstr) == 0 {
					t.Errorf("Unexpected failure: %v", err)
				} else if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("Error does not contain expected substring: %v", err)
				}
			} else {
				if len(tc.errSubstr) != 0 {
					t.Errorf("Unexpected success")
				}
			}
		})
	}
}

// TestValidateTargets tests the validateTargets function.
func TestValidateTargets(t *testing.T) {
	for _, tc := range []struct {
		name      string   // Name of test case.
		targets   []target // Initialized target structs.
		errSubstr string   // Substring expected to be in error string.
	}{{
		name:      "NoTargets",
		errSubstr: "missing: targets",
	}, {
		name: "KindNotSupported",
		targets: []target{{
			Kind:      kindDaemonSet,
			Namespace: "Namespace",
			Name:      "Name",
			Mode:      modeAllOfThem,
		}, {
			Kind:      "Job",
			Namespace: "Namespace",
			Name:      "Name",
			Mode:      modeAllOfThem,
		}},
		errSubstr: "target[1].kind invalid: Job",
	}, {
		name: "ModeNotSupported",
		targets: []target{{
			Kind:      kindDaemonSet,
			Namespace: "Namespace",
			Name:      "Name",
			Mode:      "AtLeastTwo",
		}},
		errSubstr: "target[0].mode invalid: AtLeastTwo",
	}, {
		name: "KindEmpty",
		targets: []target{{
			Kind:      "",
			Namespace: "Namespace",
			Name:      "Name",
			Mode:      modeAtLeastOne,
		}},
		errSubstr: "missing: target[0].kind",
	}, {
		name: "NameEmpty",
		targets: []target{{
			Kind:      kindDeployment,
			Namespace: "Namespace",
			Name:      "",
			Mode:      modeAtLeastOne,
		}},
		errSubstr: "missing: target[0].name",
	}, {
		name: "NamespaceEmpty",
		targets: []target{{
			Kind:      kindDeployment,
			Namespace: "",
			Name:      "Name",
			Mode:      modeAtLeastOne,
		}},
		errSubstr: "missing: target[0].namespace",
	}, {
		name: "ModeEmpty",
		targets: []target{{
			Kind:      kindDeployment,
			Namespace: "Namespace",
			Name:      "Name",
			Mode:      "",
		}},
		errSubstr: "missing: target[0].mode",
	}} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTargets(tc.targets)
			if err != nil {
				if len(tc.errSubstr) == 0 {
					t.Errorf("Unexpected failure: %v", err)
				} else if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("Error does not contain expected substring: %v", err)
				}
			} else {
				if len(tc.errSubstr) != 0 {
					t.Errorf("Unexpected success")
				}
			}
		})
	}
}
