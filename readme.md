# ONDATRA Tests for Open Traffic Generator

This repository consists of Open Traffic Generator tests written in [gosnappi](https://pkg.go.dev/github.com/open-traffic-generator/snappi/gosnappi) utilizing [ONDATRA](https://github.com/openconfig/ondatra).

### QuickStart

1. Deploy a clean Ubuntu 20.04 LTS Server with at least:
   - 8GB RAM
   - 4 CPU cores
   - 64GB Persistent Storage

2. Clone this repository and setup testbed. At the end of this step, you should have:
   - A kind cluster deployed with meshnet-cni, metallb and Ixia-c operator configured
   - Ready-to-execute test (with all the libraries built)

   ```sh
   git clone --recurse-submodules https://github.com/open-traffic-generator/ondatra-tests.git
   cd ondatra-tests && ./do.sh setup_testbed
   ```

   > You may be prompted to logout, login again re-execute same command

3. Execute a sample test

   ```sh
   # create topology if it does not exist
   ./do.sh newtop
   # run test
   ./do.sh test tests/bgp_route_install_test.go
   # delete topology if not needed anymore
   ./do.sh rmtop
   ```
