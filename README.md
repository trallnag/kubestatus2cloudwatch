# KubeStatus2CloudWatch

Small app written in Go that continuously watches the status of certain
resources in a Kubernetes cluster, aggregates these into a single value, and
uses that to update a metric in Amazon CloudWatch.

The metric will have the value 1 if all targets are health, the value 0 if at
least one target is unhealthy (according to the configuration), and missing data
if KubeStatus2CloudWatch is unhealthy / down.

## Motivation

Lately I've been using Amazon EKS for running and orchestrating containerized
workloads. To monitor the clusters and the workloads within them the popular
tools Prometheus, Grafana, and friends are used. They are hosted within the
clusters and they will notify me and my team if an alert fires.

But what if the observability system itself goes down? We won't get any
notification in that case. And since it is all self-hosted and self-contained
there are no SLAs or similar.

We somehow have to monitor the monitoring system. This is where
KubeStatus2CloudWatch comes in. It scans the status of the monitoring components
in the cluster and manages a CloudWatch metric that reflects the overall status.
Now I can go ahead and create a CloudWatch alarm and friends to monitor this one
metric.

I am also interested in learning Go and things related to it. So I took this all
as an excuse to get my hands dirty.

## Use Case

Here is a high-level overview of the use case described in
[Motivation](#motivation). KubeStatus2CloudWatch is used as a bridge between
Kubernetes and CloudWatch. The alarm fires if the metric falls below 1 or is
missing data for a certain amount of time.

[![assets/use-case-example.drawio.svg](assets/use-case-example.drawio.svg)](assets/use-case-example.drawio.svg)

This shows that KubeStatus2CloudWatch caters to a very specific use case and
must be combined with other tools to be useful.

## Getting Started

KubeStatus2CloudWatch is written in Go and the code ends up in a single
executable binary. There are three approaches:

1. Use the provided container images hosted on Docker Hub
   [here](https://hub.docker.com/r/trallnag/kubestatus2cloudwatch).
1. Get the binaries from the respective release artifacts
   [here](https://github.com/trallnag/kubestatus2cloudwatch/releases).
1. Build the binary with `go build` as usual with Go.

Create a configuration file for KubeStatus2CloudWatch. Read
[Configuration](#configuration) for more information.

The general approach:

1. Run KubeStatus2CloudWatch locally to make sure it fits your use case. Also
   write the configuration for KubeStatus2CloudWatch before going into the
   cluster.
1. Setup the IAM role in AWS. This involves a trust policy that works with IRSA
   and a policy that gives permission to the CloudWatch API.
1. Setup Kubernetes resources to get the deployment itself to work properly.
   This includes service account, role, role binding, and config map.
1. Deploy KubeStatus2CloudWatch as a deployment.

Before getting KubeStatus2CloudWatch to run in the cluster, we will first run it
locally. This requires AWS and Kubernetes credentials to be setup.

Place the `config.yaml` you have adjusted to your requirements next to the
binary. Now execute the binary. You should see in the logs that
KubeStatus2CloudWatch reads the configuration, configures things, and then
starts to scan the targets periodically and update the CloudWatch metric. There
should be no errors visible. Check the metric in CloudWatch. If everything looks
fine, you can proceed with deploying KubeStatus2CloudWatch in Kubernetes.

Note that KubeStatus2CloudWatch interacts with the Kubernetes and CloudWatch
APIs, which requires appropriate permissions. IAM Roles for Service Accounts
(IRSA) is expected to be available in the cluster.

We need a **IAM role** with the following **trust policy**:

```json
{
  "Version" : "2012-10-17",
  "Statement" : [
    {
      "Effect" : "Allow",
      "Principal" : {
        "Federated" : "arn:aws:iam::${ACCOUNT_ID}:oidc-provider/${ISSUER_URL}"
      },
      "Action" : "sts:AssumeRoleWithWebIdentity",
      "Condition" : {
        "StringEquals" : {
          "${ISSUER_URL}:aud" : "sts.amazonaws.com",
          "${ISSUER_URL}:sub" : "system:serviceaccount:${KUBE_NAMESPACE}:kubestatus2cloudwatch"
        }
      }
    }
  ]
}
```

The **inline policy** should look like this:

```json
{
  "Version" : "2012-10-17",
  "Statement" : [
    {
      "Effect" : "Allow",
      "Action" : "cloudwatch:PutMetricData",
      "Resource" : "*",
      "Condition" : {
        "StringEquals" : {
          "cloudwatch:namespace" : "${KUBE_NAMESPACE}"
        }
      }
    }
  ]
}
```

Within Kubernetes, the required **service account** references the IAM role:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  namespace: "${KUBE_NAMESPACE}"
  name: kubestatus2cloudwatch
  labels:
    app.kubernetes.io/name: kubestatus2cloudwatch
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::${ACCOUNT_ID}:role/${IAM_ROLE_NAME}
    eks.amazonaws.com/sts-regional-endpoints: "true"
```

But a **role** is also required:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: "${KUBE_NAMESPACE}"
  name: kubestatus2cloudwatch
  labels:
    app.kubernetes.io/name: kubestatus2cloudwatch
rules:
  - apiGroups: [apps]
    resources: [daemonsets, statefulsets, deployments]
    verbs: [get]
```

A **role binding** is used to associate the role with the service account:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  namespace: "${KUBE_NAMESPACE}"
  name: kubestatus2cloudwatch
  labels:
    app.kubernetes.io/name: kubestatus2cloudwatch
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: kubestatus2cloudwatch
subjects:
  - kind: ServiceAccount
    name: kubestatus2cloudwatch
    namespace: "${KUBE_NAMESPACE}"
```

To provide the configuration file to KubeStatus2CloudWatch, a **config map** is
used:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  namespace: "${KUBE_NAMESPACE}"
  name: kubestatus2cloudwatch
  labels:
    app.kubernetes.io/name: kubestatus2cloudwatch
data:
  config.yaml: |
    ...
```

Now finish it up by creating the **deployment**:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: "${KUBE_NAMESPACE}"
  name: kubestatus2cloudwatch
  labels:
    app.kubernetes.io/name: kubestatus2cloudwatch
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: kubestatus2cloudwatch
  template:
    metadata:
      labels:
        app.kubernetes.io/name: kubestatus2cloudwatch
    spec:
      containers:
        - image: trallnag/kubestatus2cloudwatch:${VERSION}
          name: kubestatus2cloudwatch
          volumeMounts:
            - name: config
              subPath: config.yaml
              mountPath: /app/config.yaml
              readOnly: true
      serviceAccountName: kubestatus2cloudwatch
      volumes:
        - name: config
          configMap:
            name: kubestatus2cloudwatch
```

Check the pod logs and the CloudWatch metric to see if things work as expected.

## Configuration

KubeStatus2CloudWatch is configured with a YAML file that is called
`config.yaml` and placed right next to binary. The app will crash during startup
without a valid configuration file.

A valid exemplary configuration with extensive comments as documentation can be
found at [`assets/config-example.yaml`](assets/config-example.yaml). It can be
used as a starting point. The file
[`assets/config-minimal.yaml`](assets/config-minimal.yaml) contains a minimal
configuration.

As a supplement the corresponding JSON schema at
[`assets/config.schema.json`](assets/config.schema.json) can be used as well.

## Links

- CodeCov: https://app.codecov.io/gh/trallnag/kubestatus2cloudwatch
- Docker Hub: https://hub.docker.com/r/trallnag/kubestatus2cloudwatch
- Pre-commit: https://results.pre-commit.ci/repo/github/582991925
