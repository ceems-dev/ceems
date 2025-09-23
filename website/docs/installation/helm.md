---
sidebar_position: 4
---

# Helm

For k8s deployments, CEEMS can be installed using [Helm](https://helm.sh/).
The charts are available at [Helm repository](https://@ceemsOrg@.github.io/helm-charts).

The helm repository can be added using the following command:

```bash
helm repo add @ceemsOrg@ https://@ceemsOrg@.github.io/helm-charts
```

Once the repository has been successfully added, it can be installed using:

```bash
helm install -n ceems --create-namespace ceems ceems-dev/kube-ceems
```

This will create a new namespace `ceems` and install all the CEEMS components along
with Prometheus and Grafana.

More instructions on how to use chart values can be found in the chart's
[README](https://github.com/@ceemsOrg@/helm-charts/blob/main/charts/kube-ceems/README.md)
