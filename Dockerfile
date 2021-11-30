FROM ubuntu:20.04
ENV SRC_ROOT=/home/ondatra-tests
ENV GOPATH=/root/go
ENV PATH=${PATH}:/usr/local/bin:/usr/local/go/bin:/root/go/bin
RUN mkdir -p ${SRC_ROOT}
# Get project source, install dependencies and build it
COPY . ${SRC_ROOT}/
RUN cd ${SRC_ROOT} && chmod +x do.sh && ./do.sh setup
RUN cd ${SRC_ROOT} && make build
WORKDIR ${SRC_ROOT}
CMD ["/bin/bash"]
