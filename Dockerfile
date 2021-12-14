FROM ubuntu:20.04
ENV SRC_ROOT=/home/ondatra-tests
RUN mkdir -p ${SRC_ROOT}
# Get project source, install dependencies and build it
COPY . ${SRC_ROOT}/
RUN cd ${SRC_ROOT} && ./do.sh setup_ondatra_tests
WORKDIR ${SRC_ROOT}
CMD ["/bin/bash"]
