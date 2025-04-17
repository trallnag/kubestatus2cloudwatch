package main

import (
	"fmt"
	"os"
	"slices"

	"gopkg.in/yaml.v3"
)

// Interval specficiation.
const (
	minSeconds     = 1
	defaultSeconds = 60
)

// Allowed logging levels.
const (
	logLevelDebug = "debug"
	logLevelInfo  = "info"
)

// Allowed logging formats.
const (
	logFormatJSON   = "json"
	logFormatLogfmt = "logfmt"
)

// logging configures logging.
type logging struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// dimension is a single CloudWatch metric dimension.
type dimension struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// metric configures the CloudWatch metric.
type metric struct {
	Namespace  string      `yaml:"namespace"`
	Name       string      `yaml:"name"`
	Dimensions []dimension `yaml:"dimensions"`
}

// Allowed target modes.
const (
	modeAllOfThem  = "AllOfThem"
	modeAtLeastOne = "AtLeastOne"
)

// Allowed target kinds.
const (
	kindDaemonSet   = "DaemonSet"
	kindDeployment  = "Deployment"
	kindStatefulSet = "StatefulSet"
)

// target is a single Kubernetes target to scan.
type target struct {
	Kind      string `yaml:"kind"`
	Namespace string `yaml:"namespace"`
	Name      string `yaml:"name"`
	Mode      string `yaml:"mode"`
}

// config is the central configuration.
// Use NewConfig to create a new config.
type config struct {
	DryRun  bool     `yaml:"dryRun"`
	Seconds int      `yaml:"seconds"`
	Metric  metric   `yaml:"metric"`
	Targets []target `yaml:"targets"`
	Logging logging  `yaml:"logging"`
}

// newConfig reads and processes the configuration.
func newConfig(configPath string) (config, error) {
	config := config{} //nolint:exhaustruct // Config is populated from file.

	configFile, err := os.ReadFile(configPath)
	if err != nil {
		return config, fmt.Errorf("read config: %v", err)
	}

	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		return config, fmt.Errorf("unmarshal config: %v", err)
	}

	config, err = processConfig(config)
	if err != nil {
		return config, fmt.Errorf("process config: %v", err)
	}

	return config, nil
}

// processConfig processes the configuration and validates it.
// It sets default values and checks for errors.
func processConfig(config config) (config, error) {
	if config.Seconds < minSeconds {
		config.Seconds = defaultSeconds
	}

	if err := validateMetric(config.Metric); err != nil {
		return config, fmt.Errorf("validate metric config: %v", err)
	}

	if err := validateTargets(config.Targets); err != nil {
		return config, fmt.Errorf("validate targets config: %v", err)
	}

	allowedLogLevels := []string{logLevelDebug, logLevelInfo}

	if config.Logging.Level == "" {
		config.Logging.Level = logLevelInfo
	} else if !slices.Contains(allowedLogLevels, config.Logging.Level) {
		return config, fmt.Errorf(
			"logging.level invalid: %v", config.Logging.Level,
		)
	}

	allowedLogFormats := []string{logFormatJSON, logFormatLogfmt}

	if config.Logging.Format == "" {
		config.Logging.Format = logFormatJSON
	} else if !slices.Contains(allowedLogFormats, config.Logging.Format) {
		return config, fmt.Errorf(
			"logging.format invalid: %v", config.Logging.Format,
		)
	}

	return config, nil
}

// validateMetric validates the metric configuration.
func validateMetric(metric metric) error {
	if metric.Namespace == "" {
		return fmt.Errorf("missing: metric.namespace")
	}

	if metric.Name == "" {
		return fmt.Errorf("missing: metric.name")
	}

	for i, dimension := range metric.Dimensions {
		if dimension.Name == "" {
			return fmt.Errorf(
				"missing: metric.dimensions[%v].name", i,
			)
		}

		if dimension.Value == "" {
			return fmt.Errorf(
				"missing: metric.dimensions[%v].value", i,
			)
		}
	}

	return nil
}

// validateTargets validates the targets configuration.
func validateTargets(targets []target) error {
	if len(targets) == 0 {
		return fmt.Errorf("missing: targets")
	}

	for i, target := range targets {
		if target.Kind == "" {
			return fmt.Errorf("missing: target[%v].kind", i)
		}

		allowedTargetKinds := []string{
			kindDeployment, kindStatefulSet, kindDaemonSet,
		}
		if !slices.Contains(allowedTargetKinds, target.Kind) {
			return fmt.Errorf(
				"target[%v].kind invalid: %v", i, target.Kind,
			)
		}

		if target.Namespace == "" {
			return fmt.Errorf("missing: target[%v].namespace", i)
		}

		if target.Name == "" {
			return fmt.Errorf("missing: target[%v].name", i)
		}

		if target.Mode == "" {
			return fmt.Errorf("missing: target[%v].mode", i)
		}

		allowedTargetModes := []string{modeAllOfThem, modeAtLeastOne}
		if !slices.Contains(allowedTargetModes, target.Mode) {
			return fmt.Errorf(
				"target[%v].mode invalid: %v", i, target.Mode,
			)
		}
	}

	return nil
}
