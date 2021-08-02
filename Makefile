all: install

install:
	go install -v

prepare-test:
	echo "..."

test:
	GO111MODULE=off go get github.com/onsi/ginkgo/ginkgo
	GO111MODULE=on ginkgo -mod vendor -r -keepGoing -p -trace -randomizeAllSpecs -progress --race

fmt:
	go fmt ./...

fly:
	fly -t persi execute -c scripts/ci/run_unit.build.yml -i smbbroker=$$PWD
