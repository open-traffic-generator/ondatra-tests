# ONDATRA Tests for Open Traffic Generator

[![Project Status: Active – The project has reached a stable, usable state and is being actively developed.](https://www.repostatus.org/badges/latest/active.svg)](https://www.repostatus.org/#active)
[![license](https://img.shields.io/badge/license-MIT-green.svg)](https://en.wikipedia.org/wiki/MIT_License)

This repository consists of Open Traffic Generator tests written in [gosnappi](https://pkg.go.dev/github.com/open-traffic-generator/snappi/gosnappi) utilizing [ONDATRA](https://github.com/openconfig/ondatra).

> NOTE: This repository is very much work in progress, hence:
> - Sending configuration to and fetching stats from DUT in tests is not done using gNMI
> - Template files (containing container image locations) for some DUTs are missing (e.g. cisco, juniper, nokia, frr, etc.)
> - Tests themselves are subject to change

### Prerequisites

1. Deploy a clean Ubuntu 20.04 LTS Server with at least:
   - 16GB RAM
   - 6-8 CPU cores
   - 128GB Persistent Storage

2. Ensure you have a valid Github account and (optionally) GCP account

3. None of the steps below should be executed as a sudo. The script will automatically prompt for:
   - sudo password when needed
   - Github or GCP credentials when needed

4. Patience - since building and running tests might take longer than usual the first time (due to large number of generated code inside ondatra)

### QuickStart

1. Clone this repository and setup testbed. At the end of this step, you should have:
   - A kind cluster deployed with meshnet-cni, metallb and Ixia-c operator configured
   - Ready-to-execute tests (with all the libraries built)

   ```sh
   git clone --recurse-submodules https://github.com/open-traffic-generator/ondatra-tests.git
   cd ondatra-tests && ./do.sh setup
   ```

   > You may be prompted to logout, login and re-execute the same command again.

2. Load Ixia-c and DUT images

   ```sh
   ./do.sh setup_repo ${repo} ${vendor}
   ```

   Examples:
      - Obtain images over GCP (requires a valid GCP account with access to project [kt-nts-athena-dev](https://console.cloud.google.com/home/dashboard?project=kt-nts-athena-dev))

      ```sh
      ./do.sh setup_repo gcp arista
      ```

      - Or, obtain images over docker.io and ghcr.io (requires a valid github account with [PAT](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token))

      ```sh
      ./do.sh setup_repo ghcr arista
      ```

3. Execute a sample test.  
   Operations below are performed based on contents of `resources/global/knebind-config.yaml` which can be changed (e.g. when a different topology config is needed).

   ```sh
   # create topology if it does not exist
   ./do.sh newtop
   # run test
   ./do.sh run tests/bgp_route_install_test.go
   # delete topology if not needed anymore
   ./do.sh rmtop
   ```

### Diagnostics

   ```sh
   # To execute any arbitrary function inside the scripts
   ./do.sh <func name>

   # list logs from all pods (for a given deployment, across multiple pod restarts)
   docker exec -it kind-control-plane ls -lht /var/log/containers
   # capture log from a given pod
   kubectl logs -n ixia-c <pod-name> <container-name> >> out.log
   # capture resource usage in case of pod crashes
   kubectl describe nodes >> out.log
   # capture pod specific issues
   kubectl -n ixia-c describe pod <pod-name> >> out.log
   # capture service specific issues
   kubectl -n ixia-c describe service <service-name> >> out.log
   ```

### What is done as part of setup ?

- Install basic utilities as prerequisites (e.g. wget, unzip, etc.)
- Install docker, go and kind
- Create a single-node kuberenetes cluster using kind and wait for it to be ready
  - If it exists already, remove it
  - The entire cluster lives inside a single docker container, so it's easy to remove
- Copy kubectl from kind cluster to host (to ensure we're using a version of kubectl that's compatible with kube api server)
- Deploy meshnet to allow point-to-point networking between pods
- Identify IP subnet for kind cluster and deploy metallb exposing public IP belonging to that subnet (for most Ixia-c services)
  - This is needed for ONDATRA's testbed reservation logic to deduce "reachable" service IP and TCP port
- Deploy Ixia-c operator from OTG repository to allow managing lifecycle of Ixia-c pods
- Install protoc for code generation in ondatra
- Install KNE to allow creating / deleting topologies (required by ondatra as well)
- Build ondatra
- Copy local `.kube/config` to a location expected by ondatra

```sh
# to only create cluster and skip test setup, execute following
./do.sh setup_cluster
# to only setup test and skip creating cluster, execute following
./do.sh setup_test_client
```

### What is done as part of setup_repo ?

- Authenticate container registry using gcloud or github, so that images for Ixia-c pods can be downloaded
- Pull all required images for Ixia-c pods and load them inside kind cluster
- Pull DUT image and load it inside kind cluster
- Generate valid KNE topology config file to be used by default (for a given repo and vendor)
- Apply offline config map for Ixia-c operator (for a given repo)

### What is done as part of run (test) ?

- Assume all images are already loaded (using `./do.sh setup_repo ${repo} ${vendor}`)
- Assume KNE topology is already created (using `./do.sh newtop`)
- Assume following configs are correct:
  - `resources/global/kubecfg` locates Kubernetes cluster and its credentials
  - `resources/global/knebind-config.yaml` locates kne_cli, kubecfg and KNE topology config
  - `resources/topology/ixia-${vendor}-ixia.txt` specifies deployed topology inside cluster
  - `resourcs/testbed/ixia-${vendor}-ixia.txt` specifies topology required by a test
- Run test file provided along with kne-bind config file and testbed file
   - The test needs to be inside `tests/` suffixed with `_test.go` (e.g. tests/bgp_route_install_test.go)
   - Config for setting/un-setting DUT is kept inside `resources/dutconfig/${test-name}/`
   - Push `set_dut` config to DUT
   - Start test and dump logs to `logs/${test-name}.log`
   - Push `unset_dut` config to DUT

### How to add new test and a new vendor ?

- Publish your DUT image and pull it in and load it inside kind cluster

   ```sh
   docker pull ${repo/img}
   kind load docker-image ${repo/img}
   ```

- Create a vendor image file `resources/global/${repo}-${vendor}.yaml` and put the image path inside it

- Create a KNE topology file `resources/topology/*-${vendor}-*.template.txt`
   * Copy contents from existing template
   * Remember to replace vendor specific names
   * Topology can be further modified to contain any number of nodes
   * Ensure `*init.txt` file it is pointing to has correct init commands for DUT

- Create a testbed file of same name as KNE topology file and put it inside `resources/testbed`
   * Copy contents from existing file
   * Remember to replace vendor specific names
   * Ensure it has as many nodes as specified in topology file

- Generate actual topology files from template (this will contain correct Ixia-c versions and DUT image path)

   ```sh
   ./do.sh generate_topology_configs ${repo} ${vendor}
   ```

- Ensure actual files created inside `resources/topology` is correct

- Provide name of newly created topology file (actual) in `resources/global/knebind-config.yaml`

- Create topology `./do.sh newtop`

- Add a new test file inside `tests` directory
   * Copy contents of exists test
   * Change test name and test contents
   * Change set/unset dut config with intended configs (DUT configs are to be kept inside `resources/dutconfig/${test-name}`)

- Run test `./do.sh run tests/${test-name}.go`
