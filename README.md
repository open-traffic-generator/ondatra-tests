# Ondatra Open Traffic Generator Tests
A repository of tests that uses the [ondatra test framework](https://github.com/openconfig/ondatra) 
and the [open-traffic-generator gosnappi package](https://github.com/open-traffic-generator).

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
- `<kne-config-file>` e.g., [resources/kneconfig/kne-001.yaml](./resources/kneconfig/kne-001.yaml)
- `<kne-testbed-file>` e.g., [resources/testbed/testbed-001.txt](./resources/testbed/testbed-001.txt)
