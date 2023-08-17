# Build the application from source
FROM registry.fedoraproject.org/fedora-minimal AS build-stage

RUN microdnf install tar gzip make -y

WORKDIR /download
RUN curl -L -O "https://go.dev/dl/go1.21.0.linux-amd64.tar.gz" && rm -rf /usr/local/go  && \
  tar -C /usr/local -xzf /download/go1.21.0.linux-amd64.tar.gz
ENV PATH="$PATH:/usr/local/go/bin"

WORKDIR /app
COPY . .
RUN make build

FROM registry.fedoraproject.org/fedora-minimal
RUN microdnf install iproute iputils procps-ng -y && microdnf clean all
WORKDIR /
COPY --from=build-stage /app/_output/userspace-failover /usr/local/bin/userspace-failover
