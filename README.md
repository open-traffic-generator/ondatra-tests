# Ondatra + GoSnappi Tests
A repository of tests that uses the [ondatra test framework](https://github.com/openconfig/ondatra) 
and the [gosnappi open-traffic-generator API](https://github.com/open-traffic-generator).

## Getting started
In order to execute and/or develop tests you will need a running 
[test environment](./TESTENV.md).

## Ondatra framework dependencies
This step needs to be done until the openconfig/ondatra repo main branch 
fully supports the `open-traffic-generator` binding.
```
make build
```
Edit the resources/kneconfig/kne-001.yaml file to reflect the path of the repository.

## Execute tests
```
go test -timeout 30s -run ^TestBGPPolicyRouteInstallation$ tests/tests -v -config ../resources/kneconfig/kne-001.yaml -testbed ../resources/testbed/testbed-001.txt
```

### Test command line args
The sample go test command line above has specific ondatra test arguments as follows:
`<ondatra-test-args> ::= <-config> <kne-config-file> <-testbed> <kne-testbed-file>`
- `<kne-config-file>`
    ```
    # sample kne config text file
    username: admin
    password: admin
    topology: ../topology/kne-config-001.txt
    cli: ../kne/kne_cli/kne_cli
    kubecfg: /home/anbalogh/.kube/config
    ```
- `<kne-testbed-file>`
    ```json
    # sample kne testbed protobuf text file
    duts {
      id: "dut1"
      vendor: ARISTA
      ports {
        id: "port1"    
      }
      ports {
        id: "port2"    
      }
    }

    duts {
      id: "dut2"
      vendor: ARISTA
      ports {
        id: "port1"    
      }
      ports {
        id: "port2"    
      }
      ports {
        id: "port3"    
      }
    }

    ates {
    id: "ate1"
    vendor: IXIA
    ports {
        id: "port1"    
    }
    }

    ates {
    id: "ate2"
    vendor: IXIA
    ports {
        id: "port1"    
    }
    }

    ates {
    id: "ate3"
    vendor: IXIA
    ports {
        id: "port1"    
    }
    } 

    links {
    a: "dut1:port1"
    b: "ate1:port1"
    }

    links {
    a: "dut1:port2"
    b: "dut2:port1"
    }

    links {
    a: "dut2:port2"
    b: "ate2:port1"
    }

    links {
    a: "dut2:port3"
    b: "ate3:port1"
    }
    ```
