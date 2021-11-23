# Test System Setup
Details for creating an ondatra test environment.

1) review kind details on the home page
   - https://kind.sigs.k8s.io/

2) install kind on your system
   - go install sigs.k8s.io/kind@latest

3) install tools
   - sudo snap install kubectl --classic
   - sudo snap install google-cloud-sdk --classic

4) setup gcloud credentials (only needs to be done once)
   - gcloud iam service-accounts list

5) if no keyfile has been created for a service account then create one
   - https://cloud.google.com/iam/docs/creating-managing-service-account-keys#creating_service_account_keys
   - use the downloaded keyfile in the following command
     - gcloud auth activate-service-account test-instance-exp@kt-nts-athena-dev.iam.gserviceaccount.com --key-file=./ixia-pull-secret/kt-nts-athena-dev-3ba2488cc69b.json
   - the following will add credential helpers to the local config file ~/.docker/config.json
     - gcloud auth configure-docker

6) any repo clone should be done in one root directory (both github and gcloud repos)
   - git clone https://bitbucket.it.keysight.com/scm/~hashwini/athena-k8s-gcp-utils.git
   - git clone https://github.com/networkop/meshnet-cni

7) clone the kne_cli and build it
   - git clone https://github.com/google/kne/
   - cd kne/kne_cli
   - go build

8) clone the gcloud keysight repo 
   - gcloud source repos clone keysight --project=kt-nts-athena-dev

9) clone latest ondatra changes from Himanshu's forked repo
   - git clone --branch otg-exp https://github.com/hashwini-keysight/ondatra

10) create the kind cluster
    - ./athena-k8s-gcp-utils.git./init_kind.sh

11) once the cluster is created you can load/unload the ixia-c operator with
    - ./athena-k8s-gcp-utils.git/kne_init.sh
    - ./athena-k8s-gcp-utils.git/kne_shut.sh