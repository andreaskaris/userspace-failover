.PHONY: build
build:
	go build -o _output/userspace-failover

.PHONY: run
run: build
	_output/userspace-failover
