package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestNewConfig tests NewConfig.
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
		name:       "5_metric_namespace_empty",
		configFile: "config-metric-namespace-empty.yaml",
		ErrSubstr:  "missing config: metric.namespace",
		expSuccess: false,
	}, {
		name:       "6_metric_name_empty",
		configFile: "config-metric-name-empty.yaml",
		ErrSubstr:  "missing config: metric.name",
		expSuccess: false,
	}, {
		name:       "7_dimension_name_empty",
		configFile: "config-dimension-name-empty.yaml",
		ErrSubstr:  "missing config: metric.dimensions",
		expSuccess: false,
	}, {
		name:       "8_dimension_value_empty",
		configFile: "config-dimension-value-empty.yaml",
		ErrSubstr:  "missing config: metric.dimensions",
		expSuccess: false,
	}, {
		name:       "9_no_targets",
		configFile: "config-no-targets.yaml",
		ErrSubstr:  "missing config: targets",
		expSuccess: false,
	}, {
		name:       "10_kind_not_supported",
		configFile: "config-unsupported-kind.yaml",
		ErrSubstr:  "kind not supported",
		expSuccess: false,
	}, {
		name:       "11_mode_not_supported",
		configFile: "config-unsupported-mode.yaml",
		ErrSubstr:  "mode not supported",
		expSuccess: false,
	}, {
		name:       "12_target_kind_empty",
		configFile: "config-target-kind-empty.yaml",
		ErrSubstr:  "missing config: target[2].kind",
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
	}, {
		name:       "15_target_no_name",
		configFile: "config-target-no-name.yaml",
		ErrSubstr:  "missing config: target[1].name",
		expSuccess: false,
	}, {
		name:       "16_target_no_mode",
		configFile: "config-target-no-mode.yaml",
		ErrSubstr:  "missing config: target[1].mode",
		expSuccess: false,
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
