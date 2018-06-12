# check_kubernetes

Icinga/Nagios Kubernetes check. Checks state of individual objects.

For use with https://github.com/Nexinto/kubernetes-icinga.

## Usage

```bash
check_kubernetes -kubeconfig ... -type deployment -namespace kube-system -name kube-dns

```

Supported types:

* pod
* replicaset
* deployment
* daemonset
* statefulset
* node
* componentstatus
* service

## Using Icinga

Use the configuration in `examples/check_kubernetes.conf` to configure check_kubernetes as a check_command in Icinga.
