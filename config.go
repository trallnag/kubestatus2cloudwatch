package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Dimension is a single CloudWatch metric dimension.
type Dimension struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// Metric configures the CloudWatch metric.
type Metric struct {
	Namespace  string      `yaml:"namespace"`
	Name       string      `yaml:"name"`
	Dimensions []Dimension `yaml:"dimensions"`
}

const KindDaemonSet = "DaemonSet"
const KindDeployment = "Deployment"
const KindStatefulSet = "StatefulSet"

const ModeAllOfThem = "AllOfThem"
const ModeAtLeastOne = "AtLeastOne"

// Target is a single Kubernetes target to scan.
type Target struct {
	// Type of target. Supported: "Deployment", "StatefulSet", and "DaemonSet".
	Kind string `yaml:"kind"`

	// Namespace of the target resource.
	Namespace string `yaml:"namespace"`

	// Name of the target resource.
	Name string `yaml:"name"`

	// Mode used for scanning target. "AllOfThem" requires all replicas to
	// be ready. "AtLeastOne" only requires at least one replica to be ready.
	Mode string `yaml:"mode"`
}

// Config is the central configuration. Use NewConfig to create a new config.
type Config struct {
	Seconds int      `yaml:"seconds"`
	Dry     bool     `yaml:"dry"`
	Metric  Metric   `yaml:"metric"`
	Targets []Target `yaml:"targets"`
	Logging struct {
		Level  string `yaml:"level"`
		Pretty bool   `yaml:"pretty"`
	} `yaml:"logging"`
}

// NewConfig reads configuration from provided file and performs checks.
func NewConfig(configPath string) (Config, error) {
	c := Config{}

	configFile, err := os.ReadFile(configPath)
	if err != nil {
		return c, fmt.Errorf("read config: %w", err)
	}

	err = yaml.Unmarshal(configFile, &c)
	if err != nil {
		return c, fmt.Errorf("unmarshal config: %w", err)
	}

	// Config.Seconds
	if c.Seconds < 5 {
		return c, fmt.Errorf("config seconds smaller than 5: %v", c.Seconds)
	}

	// Config.Metric.Namespace
	if c.Metric.Namespace == "" {
		return c, fmt.Errorf("missing config: metric.namespace")
	}

	// Config.Metric.Name
	if c.Metric.Name == "" {
		return c, fmt.Errorf("missing config: metric.namespace")
	}

	// Config.Metric.Dimensions[]
	for i, dimension := range c.Metric.Dimensions {
		if dimension.Name == "" {
			return c, fmt.Errorf(
				"missing config: metric.dimensions[%v].name", i,
			)
		}
		if dimension.Value == "" {
			return c, fmt.Errorf(
				"missing config: metric.dimensions[%v].value", i,
			)
		}
	}

	if err = ValidateTargets(c.Targets); err != nil {
		return c, fmt.Errorf("failed validating targets: %w", err)
	}

	// Config.Logging.Level
	if c.Logging.Level == "" {
		c.Logging.Level = "debug"
	}
	if !ContainsString([]string{"info", "debug"}, c.Logging.Level) {
		return c, fmt.Errorf(
			"logging.level not supported: %s", c.Logging.Level,
		)
	}

	return c, nil
}

// ValidateTargets validates targets. Used as part of configuration parsing.
func ValidateTargets(targets []Target) error {
	if len(targets) == 0 {
		return fmt.Errorf("missing config: targets")
	}
	for i, target := range targets {
		if target.Kind == "" {
			return fmt.Errorf("missing config: target[%v].kind", i)
		}
		allowedTargetKinds := []string{
			KindDeployment, KindStatefulSet, KindDaemonSet,
		}
		if !ContainsString(allowedTargetKinds, target.Kind) {
			return fmt.Errorf(
				"target[%v].kind not supported: %s", i, target.Kind,
			)
		}

		if target.Namespace == "" {
			return fmt.Errorf("missing config: target[%v].namespace", i)
		}

		if target.Name == "" {
			return fmt.Errorf("missing config: target[%v].name", i)
		}

		if target.Mode == "" {
			return fmt.Errorf("missing config: target[%v].mode", i)
		}
		allowedTargetModes := []string{ModeAllOfThem, ModeAtLeastOne}
		if !ContainsString(allowedTargetModes, target.Mode) {
			return fmt.Errorf(
				"target[%v].mode not supported: %s", i, target.Mode,
			)
		}
	}
	return nil
}

func ContainsString(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}
