# Metric

Spiderpool can be configured to serve [Opentelemetry](https://opentelemetry.io/) metrics.
And spiderpool metrics provide the insight of Spiderpool Agent and Spiderpool Controller.

## spiderpool controller

The metrics of spiderpool controller is set by the following pod environment:

| environment                 | description    | default |
| --------------------------- | -------------- | ------- |
| SPIDERPOOL_ENABLED_METRIC   | enable metrics | false   |
| SPIDERPOOL_METRIC_HTTP_PORT | metrics port   | 5721    |

## spiderpool agent

The metrics of spiderpool agent is set by the following pod environment:

| environment                 | description    | default |
| --------------------------- | -------------- | ------- |
| SPIDERPOOL_ENABLED_METRIC   | enable metrics | false   |
| SPIDERPOOL_METRIC_HTTP_PORT | metrics port   | 5721    |

## Get Started

### Enable Metric support

Firstly, please ensure you have installed the spiderpool and configured the CNI file, refer [install](./install.md) for details

Check the environment variable `SPIDERPOOL_ENABLED_METRIC` of the daemonset `spiderpool-agent` for whether it is already set to `true` or not.

Check the environment variable `SPIDERPOOL_ENABLED_METRIC` of deployment `spiderpool-controller` for whether it is already set to `true` or not.

```shell
kubectl -n kube-system get daemonset spiderpool-agent -o yaml
------
kubectl -n kube-system get deployment spiderpool-controller -o yaml
```

You can set one or both of them to `true`.
For example, let's enable spiderpool agent metrics by running `helm upgrade --set spiderpoolAgent.prometheus.enabled=true`.

## Metric reference

### Spiderpool Agent

Spiderpool agent exports some metrics related with IPAM allocation and release. Currently, those include:


| Name                                                      | description                                                  |
| --------------------------------------------------------- | ------------------------------------------------------------ |
| spiderpool_ipam_allocation_counts                         | Number of IPAM allocation requests that Spiderpool Agent received , prometheus type: counter |
| spiderpool_ipam_allocation_failure_counts                 | Number of Spiderpool Agent IPAM allocation failures, prometheus type: counter |
| spiderpool_ipam_allocation_rollback_failure_counts        | Number of Spiderpool Agent IPAM allocation rollback failures, prometheus type: counter |
| spiderpool_ipam_allocation_update_ippool_conflict_counts  | Number of Spiderpool Agent IPAM allocation update IPPool conflicts, prometheus type: counter |
| spiderpool_ipam_allocation_err_internal_counts            | Number of Spiderpool Agent IPAM allocation internal errors, prometheus type: counter |
| spiderpool_ipam_allocation_err_no_available_pool_counts   | Number of Spiderpool Agent IPAM allocation no available IPPool errors, prometheus type: counter |
| spiderpool_ipam_allocation_err_retries_exhausted_counts   | Number of Spiderpool Agent IPAM allocation retries exhausted errors, prometheus type: counter |
| spiderpool_ipam_allocation_err_ip_used_out_counts         | Number of Spiderpool Agent IPAM allocation IP addresses used out errors, prometheus type: counter |
| spiderpool_ipam_allocation_average_duration_seconds       | The average duration of all Spiderpool Agent allocation processes, prometheus type: gauge |
| spiderpool_ipam_allocation_max_duration_seconds           | The maximum duration of Spiderpool Agent allocation process (per-process), prometheus type: gauge |
| spiderpool_ipam_allocation_min_duration_seconds           | The minimum duration of Spiderpool Agent allocation process (per-process), prometheus type: gauge |
| spiderpool_ipam_allocation_latest_duration_seconds        | The latest duration of Spiderpool Agent allocation process (per-process), prometheus type: gauge |
| spiderpool_ipam_allocation_duration_seconds               | Histogram of IPAM allocation duration in seconds, prometheus type: histogram |
| spiderpool_ipam_allocation_average_limit_duration_seconds | The average duration of all Spiderpool Agent allocation queuing, prometheus type: gauge |
| spiderpool_ipam_allocation_max_limit_duration_seconds     | The maximum duration of Spiderpool Agent allocation queuing, prometheus type: gauge |
| spiderpool_ipam_allocation_min_limit_duration_seconds     | The minimum duration of Spiderpool Agent allocation queuing, prometheus type: gauge |
| spiderpool_ipam_allocation_latest_limit_duration_seconds  | The latest duration of Spiderpool Agent allocation queuing, prometheus type: gauge |
| spiderpool_ipam_allocation_limit_duration_seconds         | Histogram of IPAM allocation queuing duration in seconds, prometheus type: histogram |
| spiderpool_ipam_release_counts                            | Count of the number of Spiderpool Agent received the IPAM release requests, prometheus type: counter |
| spiderpool_ipam_release_failure_counts                    | Number of Spiderpool Agent IPAM release failure, prometheus type: counter |
| spiderpool_ipam_release_update_ippool_conflict_counts     | Number of Spiderpool Agent IPAM release update IPPool conflicts, prometheus type: counter |
| spiderpool_ipam_release_err_internal_counts               | Number of Spiderpool Agent IPAM releasing internal error, prometheus type: counter |
| spiderpool_ipam_release_err_retries_exhausted_counts      | Number of Spiderpool Agent IPAM releasing retries exhausted error, prometheus type: counter |
| spiderpool_ipam_release_average_duration_seconds          | The average duration of all Spiderpool Agent release processes, prometheus type: gauge |
| spiderpool_ipam_release_max_duration_seconds              | The maximum duration of Spiderpool Agent release process (per-process), prometheus type: gauge |
| spiderpool_ipam_release_min_duration_seconds              | The minimum duration of Spiderpool Agent release process (per-process), prometheus type: gauge |
| spiderpool_ipam_release_latest_duration_seconds           | The latest duration of Spiderpool Agent release process (per-process), prometheus type: gauge |
| spiderpool_ipam_release_duration_seconds                  | Histogram of IPAM release duration in seconds, prometheus type: histogram |
| spiderpool_ipam_release_average_limit_duration_seconds    | The average duration of all Spiderpool Agent release queuing, prometheus type: gauge |
| spiderpool_ipam_release_max_limit_duration_seconds        | The maximum duration of Spiderpool Agent release queuing, prometheus type: gauge |
| spiderpool_ipam_release_min_limit_duration_seconds        | The minimum duration of Spiderpool Agent release queuing, prometheus type: gauge |
| spiderpool_ipam_release_latest_limit_duration_seconds     | The latest duration of Spiderpool Agent release queuing, prometheus type: gauge |
| spiderpool_ipam_release_limit_duration_seconds            | Histogram of IPAM release queuing duration in seconds, prometheus type: histogram |
| spiderpool_auto_pool_waited_for_available_counts          | Number of Spiderpool Agent IPAM allocation wait for auto-created IPPool available, prometheus type: counter |

### Spiderpool Controller

Spiderpool controller exports some metrics related with SpiderIPPool IP garbage collection. Currently, those include:

| Name                                                   | description                                                  |
| ------------------------------------------------------ | ------------------------------------------------------------ |
| spiderpool_ip_gc_counts                                | Number of Spiderpool Controller IP garbage collection, prometheus type: counter |
| spiderpool_ip_gc_failure_counts                        | Number of Spiderpool Controller IP garbage collection failures, prometheus type: counter |
| spiderpool_subnet_ippool_counts                        | Number of SpiderSubnet corresponding IPPools number, prometheus type: gauge |
| spiderpool_ippool_informer_conflict_counts             | Number of Spiderpool Controller IPPool object status update operation conflict number, prometheus type: counter |
| spiderpool_auto_pool_creation_average_duration_seconds | The average duration of All new auto-created IPPools creation and mark duration, prometheus type: gauge |
| spiderpool_auto_pool_creation_max_duration_seconds     | The maximum duration of auto-created IPPool creation and mark duration (per-process), prometheus type: gauge |
| spiderpool_auto_pool_creation_min_duration_seconds     | The minimum duration of auto-created IPPool creation and mark duration (per-process), prometheus type: gauge |
| spiderpool_auto_pool_creation_latest_duration_seconds  | The latest duration of auto-created IPPool creation and mark duration (per-process), prometheus type: gauge |
| spiderpool_auto_pool_scale_average_duration_seconds    | The average duration of All auto-created IPPools scale duration, prometheus type: gauge |
| spiderpool_auto_pool_scale_max_duration_seconds        | The maximum duration of auto-created IPPool scale duration (per-process), prometheus type: gauge |
| spiderpool_auto_pool_scale_min_duration_seconds        | The minimum duration of auto-created IPPool scale duration (per-process), prometheus type: gauge |
| spiderpool_auto_pool_scale_latest_duration_seconds     | The latest duration of auto-created IPPool scale duration (per-process), prometheus type: gauge |
| spiderpool_auto_pool_scale_duration_seconds            | Histogram of new auto-created IPPool scale duration in seconds, prometheus type: histogram |
| spiderpool_auto_pool_scale_conflict_counts             | Number of Spiderpool Controller auto-created IPPool scale operation conflicts, prometheus type: counter |
