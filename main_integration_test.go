package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/k3s"
	"gopkg.in/yaml.v3"
)

func TestIntegrationRunMain(t *testing.T) {
	if testing.Short() {
		t.Skip("Short mode requested. Skipping integration test.")
	}

	tempDir := t.TempDir()

	ctx, cancelK3s := context.WithDeadline(t.Context(), time.Now().Add(5*time.Minute))
	defer cancelK3s()

	k3sContainer, err := k3s.Run(ctx, "docker.io/rancher/k3s:v1.27.1-k3s1")
	testcontainers.CleanupContainer(t, k3sContainer)

	if err != nil {
		t.Fatal(err)
	}

	kubeConfigYaml, err := k3sContainer.GetKubeConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("kubeConfigYaml: %s", kubeConfigYaml)

	kubeConfigFile, err := os.CreateTemp(tempDir, "kubeconfig.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := kubeConfigFile.Write(kubeConfigYaml); err != nil {
		t.Fatal(err)
	}

	if err := kubeConfigFile.Close(); err != nil {
		t.Fatal(err)
	}

	t.Setenv("KUBECONFIG", kubeConfigFile.Name())

	originalArgs := os.Args

	defer func() {
		os.Args = originalArgs
	}()

	os.Args = []string{"kubestatus2cloudwatch"}

	config, err := NewConfig("assets/config-minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}

	config.Seconds = 1

	configYaml, err := yaml.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}

	configFile, err := os.CreateTemp(tempDir, "config.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := configFile.Write(configYaml); err != nil {
		t.Fatal(err)
	}

	if err := configFile.Close(); err != nil {
		t.Fatal(err)
	}

	t.Setenv("KS2CW_CONFIG_PATH", configFile.Name())

	RunMain()
}
