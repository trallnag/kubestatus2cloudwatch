[![status](https://img.shields.io/badge/status-active-brightgreen)](#project-status)
[![release](https://img.shields.io/github/v/release/trallnag/kubestatus2cloudwatch)](https://github.com/trallnag/kubestatus2cloudwatch/releases)
[![ci](https://img.shields.io/github/actions/workflow/status/trallnag/kubestatus2cloudwatch/ci.yaml?label=ci)](https://github.com/trallnag/kubestatus2cloudwatch/actions/workflows/ci.yaml)
[![release](https://img.shields.io/github/actions/workflow/status/trallnag/kubestatus2cloudwatch/release.yaml?label=release)](https://github.com/trallnag/kubestatus2cloudwatch/actions/workflows/release.yaml)

# Kubestatus2cloudwatch <!-- omit from toc -->

Small program written in Go that continuously watches the status of certain
resources in a Kubernetes cluster, aggregates these into a single value, and
uses that to update a metric in Amazon CloudWatch.

The metric will have the value 1 if all targets are healthy, the value 0 if at
least one target is unhealthy (according to the configuration), and missing data
if Kubestatus2cloudwatch is unhealthy / down.

## Table of contents <!-- omit from toc -->

- [Motivation](#motivation)
- [Use case](#use-case)
- [Getting started](#getting-started)
- [Configuration](#configuration)
- [Project status](#project-status)
- [Versioning](#versioning)
- [Contributing](#contributing)
- [Licensing](#licensing)

## Motivation

Lately (2022) I've been using Amazon EKS for running and orchestrating
containerized workloads. To monitor the clusters and the workloads within them
the popular tools Prometheus, Grafana, and friends are used. They are hosted
within the clusters and they will notify my team and me if an alert fires.

But what if the observability system itself goes down? We won't get any
notification in that case. And since it is all self-hosted and self-contained
there are no SLAs or similar.

We somehow have to monitor the monitoring system. This is where
Kubestatus2cloudwatch comes in. It scans the status of the monitoring components
in the cluster and manages a CloudWatch metric that reflects the overall status.
Now I can go ahead and create a CloudWatch alarm and friends to monitor this one
metric.

I am also interested in learning Go and things related to it.

## Use case

Here is a high-level overview of the use case described in
[Motivation](#motivation). Kubestatus2cloudwatch is used as a bridge between
Kubernetes and CloudWatch. The alarm fires if the metric falls below 1 or is
missing data for a certain amount of time.

[![assets/use-case-example.drawio.svg](./assets/use-case-example.drawio.svg)](./assets/use-case-example.drawio.svg)

Kubestatus2cloudwatch caters to a specific use case and must be combined with
other tools to be useful.

## Getting started

Kubestatus2cloudwatch is written in Go and the code ends up in a single
executable binary. There are three approaches:

1. Use the provided container images hosted on GitHub Packages
   [here](https://github.com/trallnag/kubestatus2cloudwatch/pkgs/container/kubestatus2cloudwatch).
1. Get the binaries from the respective release artifacts
   [here](https://github.com/trallnag/kubestatus2cloudwatch/releases).
1. Build the binary with `go build` as usual with Go.

Create a configuration file for Kubestatus2cloudwatch. Read
[Configuration](#configuration) for more information.

The general approach:

1. Run Kubestatus2cloudwatch locally to make sure it fits your use case. Also
   write the configuration for Kubestatus2cloudwatch before going into the
   cluster.
1. Setup the IAM role in AWS. This involves a trust policy that works with IRSA
   and a policy that gives permission to the CloudWatch API.
1. Setup Kubernetes resources to get the deployment itself to work properly.
   This includes service account, role, role binding, and config map.
1. Deploy Kubestatus2cloudwatch as a deployment.

Before getting Kubestatus2cloudwatch to run in the cluster, we will first run it
locally. This requires AWS and Kubernetes credentials to be setup.

Place the `config.yaml` you have adjusted to your requirements next to the
binary. Now execute the binary. You should see in the logs that
Kubestatus2cloudwatch reads the configuration, configures things, and then
starts to scan the targets periodically and update the CloudWatch metric. There
should be no errors visible. Check the metric in CloudWatch. If everything looks
fine, you can proceed with deploying Kubestatus2cloudwatch in Kubernetes.

Note that Kubestatus2cloudwatch interacts with the Kubernetes and CloudWatch
APIs, which requires appropriate permissions. IAM Roles for Service Accounts
(IRSA) is expected to be available in the cluster.

We need a **IAM role** with the following **trust policy**:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::${ACCOUNT_ID}:oidc-provider/${ISSUER_URL}"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "${ISSUER_URL}:aud": "sts.amazonaws.com",
          "${ISSUER_URL}:sub": "system:serviceaccount:${KUBE_NAMESPACE}:kubestatus2cloudwatch"
        }
      }
    }
  ]
}
```

The **inline policy** should look like this:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "cloudwatch:PutMetricData",
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "cloudwatch:namespace": "${KUBE_NAMESPACE}"
        }
      }
    }
  ]
}
```

Within Kubernetes, the required **Service Account** references the IAM role:

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

But a **Role** is also required:

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

A **Role Binding** is used to associate the Role with the Service Account:

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

To provide the configuration file to Kubestatus2cloudwatch, a **Config Map** is
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

Now finish it up by creating the **Deployment**:

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
        - image: ghcr.io/trallnag/kubestatus2cloudwatch:${VERSION}
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

Check the container logs and the CloudWatch metric to see if things work as
expected.

## Configuration

Kubestatus2cloudwatch is configured with a YAML file that is called
`config.yaml` and placed right next to binary. The app will crash during startup
without a valid configuration file.

A valid exemplary configuration with extensive comments as documentation can be
found at [`assets/config-example.yaml`](./assets/config-example.yaml). It can be
used as a starting point. The file
[`assets/config-minimal.yaml`](./assets/config-minimal.yaml) contains a minimal
configuration.

As a supplement the corresponding JSON schema at
[`assets/config.schema.json`](./assets/config.schema.json) can be used as well.

## Project status

The project is maintained by me, [Tim](https://github.com/trallnag), and I am
interested in keeping it alive as I am actively using it.

## Versioning

The project follows [Semantic Versioning](https://semver.org/).

## Contributing

Contributions are welcome. Please refer to [`CONTRIBUTE.md`](./CONTRIBUTE.md).

## Licensing

This work is licensed under the
[ISC license](https://en.wikipedia.org/wiki/ISC_license). See
[`LICENSE`](./LICENSE) for the license text.
