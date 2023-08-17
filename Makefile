IMAGE ?= quay.io/akaris/userspace-failover

.PHONY: clean
clean:
	rm -f _output/*

.PHONY: build
build: clean
	go build -buildvcs=false -o _output/userspace-failover

.PHONY: run
run: build
	_output/userspace-failover

.PHONY: e2e-test
e2e-test: build
	./test.sh

.PHONY: container-build
container-build: clean
	podman build -t $(IMAGE) .

.PHONY: container-push
container-push:
	podman push $(IMAGE)
