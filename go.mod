module code.cloudfoundry.org/smbbroker

require (
	code.cloudfoundry.org/clock v0.0.0-20180518195852-02e53af36e6c
	code.cloudfoundry.org/debugserver v0.0.0-20180612203758-a3ba348dfede
	code.cloudfoundry.org/existingvolumebroker v0.3.0
	code.cloudfoundry.org/goshims v0.1.0
	code.cloudfoundry.org/lager v2.0.0+incompatible
	code.cloudfoundry.org/service-broker-store v0.1.0
	code.cloudfoundry.org/volume-mount-options v0.1.0
	github.com/google/gofuzz v1.1.0
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d // indirect
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/pivotal-cf/brokerapi v2.0.5+incompatible
	github.com/tedsuo/ifrit v0.0.0-20191009134036-9a97d0632f00
	golang.org/x/crypto v0.0.0-20200302210943-78000ba7a073 // indirect
)

go 1.13
