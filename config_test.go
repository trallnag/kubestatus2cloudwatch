package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestNewConfig tests NewConfig (and indirectly all the validation functions
// that are used to reduce the complexity of NewConfig).
func TestNewConfig(t *testing.T) {
	for _, tc := range []struct {
		name       string // Name of test case.
		configFile string // Name of config file in testdata dir.
		ErrSubstr  string // Substring expected to be in error string.
		expSuccess bool   // Is NewConfig call expected to succeed?
	}{{
		name:       "1_complete",
		configFile: "config-complete.yaml",
		ErrSubstr:  "",
		expSuccess: true,
	}, {
		name:       "2_file_missing",
		configFile: "config-nonexisting.yaml",
		ErrSubstr:  "read config",
		expSuccess: false,
	}, {
		name:       "3_invalid_yaml",
		configFile: "config-invalid-yaml.yaml",
		ErrSubstr:  "unmarshal",
		expSuccess: false,
	}, {
		name:       "4_seconds_too_small",
		configFile: "config-seconds.yaml",
		ErrSubstr:  "seconds smaller",
		expSuccess: false,
	}, {
		name:       "13_invalid_log_level",
		configFile: "config-invalid-log-level.yaml",
		ErrSubstr:  "logging.level not supported",
		expSuccess: false,
	}, {
		name:       "14_log_level_default",
		configFile: "config-log-level-default.yaml",
		expSuccess: true,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := NewConfig(filepath.Join("testdata", tc.configFile))
			if tc.expSuccess && err != nil {
				t.Fatalf("Unexpected failure: %s", err.Error())
			}
			if !tc.expSuccess && err == nil {
				t.Fatal("Unexpected success.")
			}
			if !tc.expSuccess && !strings.Contains(err.Error(), tc.ErrSubstr) {
				t.Fatalf(
					"Error does not contain expected substring: got %q, want substring %q",
					err.Error(), tc.ErrSubstr,
				)
			}

			if err == nil {
				if c.Seconds < 5 {
					t.Errorf("Config seconds must be > 5: got %v", c.Seconds)
				}
			}
		})
	}
}

// TestValidateMetric tests ValidateMetric.
func TestValidateMetric(t *testing.T) {
	for _, tc := range []struct {
		name      string // Name of test case.
		metric    Metric // Initialized metric struct.
		errSubstr string // Substring expected in error string.
	}{{
		name: "1_namespace_empty",
		metric: Metric{
			Name: "Name",
		},
		errSubstr: "missing: metric.namespace",
	}, {
		name: "2_name_empty",
		metric: Metric{
			Namespace: "Namespace",
		},
		errSubstr: "missing: metric.name",
	}, {
		name: "3_dimension_name_empty",
		metric: Metric{
			Name:       "Name",
			Namespace:  "Namespace",
			Dimensions: []Dimension{{Value: "Value"}},
		},
		errSubstr: "missing: metric.dimensions[0].name",
	}, {
		name: "4_dimension_value_empty",
		metric: Metric{
			Name:       "Name",
			Namespace:  "Namespace",
			Dimensions: []Dimension{{Name: "Name"}},
		},
		errSubstr: "missing: metric.dimensions[0].value",
	}, {
		name: "5_all_is_good",
		metric: Metric{
			Name:      "Name",
			Namespace: "Namespace",
			Dimensions: []Dimension{
				{Name: "Name1", Value: "Value1"},
				{Name: "Name2", Value: "Value2"},
			},
		},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateMetric(tc.metric)
			if err != nil {
				if len(tc.errSubstr) == 0 {
					t.Errorf("Unexpected failure: %s", err.Error())
				} else if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf(
						"Err does not contain substr: got %q, want substr %q",
						err.Error(), tc.errSubstr,
					)
				}
			} else {
				if len(tc.errSubstr) != 0 {
					t.Error("Unexpected success.")
				}
			}
		})
	}
}

// TestValidateTargets tests ValidateTargets.
func TestValidateTargets(t *testing.T) {
	for _, tc := range []struct {
		name      string   // Name of test case.
		targets   []Target // Initialized target structs.
		errSubstr string   // Substring expected in error string.
	}{{
		name:      "1_no_targets",
		errSubstr: "missing: targets",
	}, {
		name: "2_kind_not_supported",
		targets: []Target{{
			Kind:      KindDaemonSet,
			Namespace: "Namespace",
			Name:      "Name",
			Mode:      ModeAllOfThem,
		}, {
			Kind:      "Job",
			Namespace: "Namespace",
			Name:      "Name",
			Mode:      ModeAllOfThem,
		}},
		errSubstr: "target[1].kind not supported: Job",
	}, {
		name: "3_mode_not_supported",
		targets: []Target{{
			Kind:      KindDaemonSet,
			Namespace: "Namespace",
			Name:      "Name",
			Mode:      "AtLeastTwo",
		}},
		errSubstr: "target[0].mode not supported: AtLeastTwo",
	}, {
		name: "4_kind_empty",
		targets: []Target{{
			Kind:      "",
			Namespace: "Namespace",
			Name:      "Name",
			Mode:      ModeAtLeastOne,
		}},
		errSubstr: "missing: target[0].kind",
	}, {
		name: "5_name_empty",
		targets: []Target{{
			Kind:      KindDeployment,
			Namespace: "Namespace",
			Name:      "",
			Mode:      ModeAtLeastOne,
		}},
		errSubstr: "missing: target[0].name",
	}, {
		name: "6_namespace_empty",
		targets: []Target{{
			Kind:      KindDeployment,
			Namespace: "",
			Name:      "Name",
			Mode:      ModeAtLeastOne,
		}},
		errSubstr: "missing: target[0].namespace",
	}, {
		name: "7_mode_empty",
		targets: []Target{{
			Kind:      KindDeployment,
			Namespace: "Namespace",
			Name:      "Name",
			Mode:      "",
		}},
		errSubstr: "missing: target[0].mode",
	}} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTargets(tc.targets)
			if err != nil {
				if len(tc.errSubstr) == 0 {
					t.Errorf("Unexpected failure: %s", err.Error())
				} else if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf(
						"Err does not contain substr: got %q, want substr %q",
						err.Error(), tc.errSubstr,
					)
				}
			} else {
				if len(tc.errSubstr) != 0 {
					t.Error("Unexpected success.")
				}
			}
		})
	}
}
