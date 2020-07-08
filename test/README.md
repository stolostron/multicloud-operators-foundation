# Mmulticloud Manager End-to-End Test

The tests contained in this repository will test the core functionality of Multicloud Management automatically with minor configurations and set up required.

## Preparation

* A Kubernetes cluster with Multicloud Management installed
* Before running testing, kubeconfig file for cluster admin should be in place. Test suites try to load the configuration file from either `$HOME/.kube/config` or the path set by environment variable `KUBECONFIG`.
* If you want to run e2e locally, when managedcluster do not deployed on hub, please set env `export  SINGLE_MANAGED_CLUSTER_ON_HUB=FALSE`.

## How to run

### Run all test suites

```sh
make e2e-test
```

Or run with a customized kubeconfig file:

```sh
make e2e-test KUBECONFIG=<path/to/file>
```

### Run a test suite

```sh
make run-e2e-test TEST_SUITE=<test_suite_name>
```

Below is the list of all available test suites:

* actions
* clusterinfos
* views

### Results

A report of testing will be generated at `/tmp/multicloud-manager-e2e-test--<datetime>.log`. No test cases should be failed if Multicloud Management is installed and configured correctly. Some test cases will be skipped if there is no managed cluster joined to the current hub.
