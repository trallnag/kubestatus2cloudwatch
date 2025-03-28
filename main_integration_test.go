package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	cw "github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	sts "github.com/aws/aws-sdk-go-v2/service/sts"
	nat "github.com/docker/go-connections/nat"
	testcontainers "github.com/testcontainers/testcontainers-go"
	k3s "github.com/testcontainers/testcontainers-go/modules/k3s"
	localstack "github.com/testcontainers/testcontainers-go/modules/localstack"
	yaml "gopkg.in/yaml.v3"
	kubeappsv1 "k8s.io/api/apps/v1"
	kubecorev1 "k8s.io/api/core/v1"
	kubemetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubewait "k8s.io/apimachinery/pkg/util/wait"
	kube "k8s.io/client-go/kubernetes"
	kubeclientcmd "k8s.io/client-go/tools/clientcmd"
)

// TestIntegrationRunMain tests behavior of app with K3s and LocalStack.
func TestIntegrationRunMain(t *testing.T) {
	if testing.Short() {
		t.Skip("Short mode requested. Skipping integration test.")
	}

	dockerHost := setUpDocker(t)

	setUpKubernetes(t)

	cloudWatchClient := setUpLocalStack(t, dockerHost)

	originalArgs := os.Args

	t.Cleanup(func() {
		os.Args = originalArgs
	})

	os.Args = []string{"kubestatus2cloudwatch"}

	config, err := newConfig("assets/config-minimal.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	config.Seconds = 1

	t.Run("Healthy", func(t *testing.T) {
		log := newLogger(t)

		config.Metric.Name = "Healthy"
		config.Targets = []target{
			{
				Kind:      "Deployment",
				Namespace: "default",
				Name:      "busybox-healthy",
				Mode:      "AllOfThem",
			},
		}

		setUpConfig(t, config)

		runMainCtx, cancelRunMain := context.WithTimeout(
			t.Context(),
			4*time.Second,
		)

		t.Cleanup(cancelRunMain)

		runMain(runMainCtx, log)

		lastMetricValue := getLastMetricValue(
			t,
			cloudWatchClient,
			config.Metric.Name,
		)
		if lastMetricValue < 1 {
			t.Fatalf("Unexpected metric value: %f", lastMetricValue)
		}
	})

	t.Run("Unhealthy", func(t *testing.T) {
		log := newLogger(t)

		config.Metric.Name = "Unhealthy"
		config.Targets = []target{
			{
				Kind:      "Deployment",
				Namespace: "default",
				Name:      "busybox-unhealthy",
				Mode:      "AllOfThem",
			},
		}

		setUpConfig(t, config)

		runMainCtx, cancelRunMain := context.WithTimeout(
			t.Context(),
			4*time.Second,
		)

		t.Cleanup(cancelRunMain)

		runMain(runMainCtx, log)

		lastMetricValue := getLastMetricValue(
			t,
			cloudWatchClient,
			config.Metric.Name,
		)
		if lastMetricValue > 0 {
			t.Fatalf("Unexpected metric value: %f", lastMetricValue)
		}
	})
}

// setUpDocker sets up the Docker provider and pulls necessary images. It
// returns the Docker host address. The images are defined in the .env file.
func setUpDocker(t *testing.T) string {
	t.Helper()

	dockerProvider, err := testcontainers.NewDockerProvider()
	if err != nil {
		t.Fatalf("Failed to create Docker provider: %v", err)
	}

	t.Cleanup(func() {
		dockerProvider.Close()
	})

	if err = dockerProvider.PullImage(t.Context(), dotEnv["BUSYBOX_IMAGE_NAME"]); err != nil {
		t.Fatalf("Failed to pull Busybox image: %v", err)
	}

	if err = dockerProvider.PullImage(t.Context(), dotEnv["KUBERNETES_IMAGE_NAME"]); err != nil {
		t.Fatalf("Failed to pull Kubernetes image: %v", err)
	}

	if err = dockerProvider.PullImage(t.Context(), dotEnv["LOCALSTACK_IMAGE_NAME"]); err != nil {
		t.Fatalf("Failed to pull LocalStack image: %v", err)
	}

	dockerHost, err := dockerProvider.DaemonHost(t.Context())
	if err != nil {
		t.Fatalf("Failed to get Docker host: %v", err)
	}

	return dockerHost
}

// setUpKubernetes starts a K3s container and sets up the Kubernetes client
// configuration. It also creates a healthy and an unhealthy BusyBox
// deployment in the default namespace.
func setUpKubernetes(t *testing.T) {
	t.Helper()

	k3sContainer, err := k3s.Run(t.Context(), dotEnv["KUBERNETES_IMAGE_NAME"])
	testcontainers.CleanupContainer(t, k3sContainer)

	if err != nil {
		t.Fatalf("Failed to start K3s container: %v", err)
	}

	kubeConfigYaml, err := k3sContainer.GetKubeConfig(t.Context())
	if err != nil {
		t.Fatalf("Failed to get Kubernetes config: %v", err)
	}

	kubeConfigFile, err := os.CreateTemp(t.TempDir(), "kubeconfig.yaml")
	if err != nil {
		t.Fatalf("Failed to create Kubernetes config file: %v", err)
	}

	if _, err := kubeConfigFile.Write(kubeConfigYaml); err != nil {
		t.Fatalf("Failed to write Kubernetes config file: %v", err)
	}

	if err := kubeConfigFile.Close(); err != nil {
		t.Fatalf("Failed to close Kubernetes config file: %v", err)
	}

	t.Setenv("KUBECONFIG", kubeConfigFile.Name())

	kubeRestConfig, err := kubeclientcmd.RESTConfigFromKubeConfig(
		kubeConfigYaml,
	)
	if err != nil {
		t.Fatalf("Failed to create Kubernetes REST config: %v", err)
	}

	kubeClient, err := kube.NewForConfig(kubeRestConfig)
	if err != nil {
		t.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	healthyBusyBoxDeployment, err := kubeClient.AppsV1().
		Deployments("default").
		Create(t.Context(), &kubeappsv1.Deployment{
			ObjectMeta: kubemetav1.ObjectMeta{
				Name: "busybox-healthy",
			},
			Spec: kubeappsv1.DeploymentSpec{
				Replicas: getInt32Ptr(t, 2),
				Selector: &kubemetav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "busybox-healthy",
					},
				},
				Template: kubecorev1.PodTemplateSpec{
					ObjectMeta: kubemetav1.ObjectMeta{
						Labels: map[string]string{
							"app": "busybox-healthy",
						},
					},
					Spec: kubecorev1.PodSpec{
						Containers: []kubecorev1.Container{
							{
								Name:  "app",
								Image: dotEnv["BUSYBOX_IMAGE_NAME"],
								Args:  []string{"sleep", "3600"},
							},
						},
					},
				},
			},
		}, kubemetav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create healthyBusyBoxDeployment: %v", err)
	}

	err = kubewait.PollUntilContextTimeout(
		t.Context(),
		1*time.Second,
		25*time.Second,
		true,
		func(ctx context.Context) (bool, error) {
			healthyBusyBoxDeployment, err = kubeClient.AppsV1().
				Deployments("default").
				Get(ctx, healthyBusyBoxDeployment.Name, kubemetav1.GetOptions{})
			if err != nil {
				return false, fmt.Errorf(
					"get healthyBusyBoxDeployment: %v",
					err,
				)
			}

			if healthyBusyBoxDeployment.Status.ReadyReplicas == 2 {
				return true, nil
			}

			t.Log("Waiting for healthyBusyBoxDeployment to be ready.")

			return false, nil
		},
	)
	if err != nil {
		t.Fatalf("Failed to get healthyBusyBoxDeployment: %v", err)
	}

	healthyBusyBoxDeployment, err = kubeClient.AppsV1().
		Deployments("default").
		Get(t.Context(), healthyBusyBoxDeployment.Name, kubemetav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get healthyBusyBoxDeployment: %v", err)
	}

	got := healthyBusyBoxDeployment.Status.ReadyReplicas
	want := int32(2)

	if got != want {
		t.Fatalf(
			"Unexpected number of ready replicas for healthyBusyBoxDeployment: got %d, want %d",
			got,
			want,
		)
	}

	unhealthyBusyBoxDeployment, err := kubeClient.AppsV1().
		Deployments("default").
		Create(t.Context(), &kubeappsv1.Deployment{
			ObjectMeta: kubemetav1.ObjectMeta{
				Name: "busybox-unhealthy",
			},
			Spec: kubeappsv1.DeploymentSpec{
				Replicas: getInt32Ptr(t, 1),
				Selector: &kubemetav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "busybox-unhealthy",
					},
				},
				Template: kubecorev1.PodTemplateSpec{
					ObjectMeta: kubemetav1.ObjectMeta{
						Labels: map[string]string{
							"app": "busybox-unhealthy",
						},
					},
					Spec: kubecorev1.PodSpec{
						Containers: []kubecorev1.Container{
							{
								Name:  "app",
								Image: dotEnv["BUSYBOX_IMAGE_NAME"],
								Args:  []string{"exit", "1"},
							},
						},
					},
				},
			},
		}, kubemetav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create unhealthyBusyBoxDeployment: %v", err)
	}

	time.Sleep(1 * time.Second)

	unhealthyBusyBoxDeployment, err = kubeClient.AppsV1().
		Deployments("default").
		Get(t.Context(), unhealthyBusyBoxDeployment.Name, kubemetav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get unhealthyBusyBoxDeployment: %v", err)
	}

	got = unhealthyBusyBoxDeployment.Status.ReadyReplicas
	want = int32(0)

	if got != want {
		t.Fatalf(
			"Unexpected number of ready replicas for unhealthyBusyBoxDeployment: got %d, want %d",
			got,
			want,
		)
	}
}

// setUpLocalStack starts a LocalStack container and sets up the AWS SDK
// configuration to use it. It also verifies the caller identity using STS.
func setUpLocalStack(
	t *testing.T,
	dockerHost string,
) *cw.Client {
	t.Helper()

	localStackContainer, err := localstack.Run(t.Context(),
		dotEnv["LOCALSTACK_IMAGE_NAME"],
		testcontainers.WithEnv(map[string]string{
			"SERVICES": "cloudwatch,iam,sts",
		}),
	)
	testcontainers.CleanupContainer(t, localStackContainer)

	if err != nil {
		t.Fatalf("Failed to start LocalStack container: %v", err)
	}

	localStackPort, err := localStackContainer.MappedPort(
		t.Context(),
		nat.Port("4566/tcp"),
	)
	if err != nil {
		t.Fatalf("Failed to get LocalStack port: %v", err)
	}

	awsEndPoint := "http://" + dockerHost + ":" + localStackPort.Port()
	t.Logf("LocalStack endpoint: %s", awsEndPoint)

	awsRegion := "eu-central-1"
	awsKey := "testKey"
	awsSecret := "testSecret"
	awsSession := "testSession"

	awsConfig, err := awsconfig.LoadDefaultConfig(t.Context(),
		awsconfig.WithRegion(awsRegion),
		awsconfig.WithCredentialsProvider(
			awscreds.NewStaticCredentialsProvider(
				awsKey,
				awsSecret,
				awsSession,
			),
		),
	)
	if err != nil {
		t.Fatalf("Failed to load AWS config: %v", err)
	}

	stsClient := sts.NewFromConfig(awsConfig, func(o *sts.Options) {
		o.BaseEndpoint = aws.String(awsEndPoint)
	})

	awsCallerIdentity, err := stsClient.GetCallerIdentity(t.Context(),
		&sts.GetCallerIdentityInput{},
	)
	if err != nil {
		t.Fatalf("Failed to get caller identity: %v", err)
	}

	awsCallerIdentityArn := aws.ToString(awsCallerIdentity.Arn)
	if awsCallerIdentityArn != "arn:aws:iam::000000000000:root" {
		t.Fatalf("Unexpected caller identity: %s", awsCallerIdentityArn)
	}

	t.Setenv("AWS_ENDPOINT_URL", awsEndPoint)
	t.Setenv("AWS_DEFAULT_REGION", awsRegion)
	t.Setenv("AWS_REGION", awsRegion)
	t.Setenv("AWS_ACCESS_KEY_ID", awsKey)
	t.Setenv("AWS_SECRET_ACCESS_KEY", awsSecret)
	t.Setenv("AWS_SESSION_TOKEN", awsSession)

	cwClient := cw.NewFromConfig(
		awsConfig,
		func(o *cw.Options) {
			o.BaseEndpoint = aws.String(awsEndPoint)
		},
	)

	return cwClient
}

// setUpConfig writes given config to a temporary file and sets up env var.
func setUpConfig(t *testing.T, config config) {
	t.Helper()

	configYaml, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	configFile, err := os.CreateTemp(t.TempDir(), "config.yaml")
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	if _, err := configFile.Write(configYaml); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	if err := configFile.Close(); err != nil {
		t.Fatalf("Failed to close config file: %v", err)
	}

	t.Setenv("KS2CW_CONFIG_PATH", configFile.Name())
}

// getLastMetricValue retrieves the last metric value from CloudWatch. The
// namespace and other options for CloudWatch are hardcoded.
func getLastMetricValue(
	t *testing.T,
	cloudWatchClient *cw.Client,
	metricName string,
) float64 {
	t.Helper()

	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Minute)

	result, err := cloudWatchClient.GetMetricStatistics(t.Context(),
		&cw.GetMetricStatisticsInput{
			Namespace:  aws.String("MyNamespace"),
			MetricName: aws.String(metricName),
			Period:     aws.Int32(60),
			StartTime:  aws.Time(startTime),
			EndTime:    aws.Time(endTime),
			Statistics: []cwtypes.Statistic{cwtypes.StatisticMaximum},
		},
	)
	if err != nil {
		t.Fatalf("Failed to get statistics: %v", err)
	}

	if len(result.Datapoints) == 0 {
		t.Fatalf("No data points found for metric: %s", metricName)
	}

	latest := result.Datapoints[0]
	for _, dp := range result.Datapoints {
		if dp.Timestamp.After(*latest.Timestamp) {
			latest = dp
		}
	}

	return *latest.Maximum
}

// getInt32Ptr returns pointer to the given value. Necessary due to Kubernetes.
func getInt32Ptr(t *testing.T, i int32) *int32 {
	t.Helper()

	return &i
}
