.PHONY: build
build:
	go build -buildvcs=false -o _output/userspace-failover

.PHONY: run
run: build
	_output/userspace-failover

.PHONY: e2e-test
e2e-test: build
	./test.sh
