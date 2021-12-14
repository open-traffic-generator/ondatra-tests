# ONDATRA Tests for Open Traffic Generator

This repository consists of Open Traffic Generator tests written in [gosnappi](https://pkg.go.dev/github.com/open-traffic-generator/snappi/gosnappi) utilizing [ONDATRA](https://github.com/openconfig/ondatra).

### QuickStart

1. Deploy a clean Ubuntu 20.04 LTS Server with at least:
   - 8GB RAM
   - 4 CPU cores
   - 64GB Persistent Storage

2. Clone this repository and setup testbed. At the end of this step, you should have:
   - A kind cluster deployed with meshnet-cni, metallb and Ixia-c operator configured
   - A docker image for test client 

   ```sh
   git clone --recurse-submodules https://github.com/open-traffic-generator/ondatra-tests.git
   cd ondatra-tests && ./do.sh setup_testbed
   ```

3. Execute a sample test

   ```sh
   # get inside test client
   docker exec -it ondatra-tests /bin/bash
   # run test
   go test -run ^TestBgpRouteInstall$ -v
   ```
