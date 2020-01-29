all: install

install:
	go install -v

test:
	GO111MODULE=off go get github.com/onsi/ginkgo/ginkgo
	GO111MODULE=on ginkgo -v -r -keepGoing -trace -randomizeAllSpecs -progress --nodes=1

fmt:
	go fmt ./... -v

fly:
	fly -t persi execute -c scripts/ci/run_unit.build.yml -i smbbroker=$$PWD