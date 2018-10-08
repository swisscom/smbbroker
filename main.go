package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/debugserver"
	"code.cloudfoundry.org/existingvolumebroker"
	"code.cloudfoundry.org/existingvolumebroker/utils"
	"code.cloudfoundry.org/goshims/osshim"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerflags"

	"code.cloudfoundry.org/service-broker-store/brokerstore"
	"github.com/pivotal-cf/brokerapi"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
)

var atAddress = flag.String(
	"listenAddr",
	"0.0.0.0:8999",
	"host:port to serve service broker API",
)

var servicesConfig = flag.String(
	"servicesConfig",
	"",
	"[REQUIRED] - Path to services config to register with cloud controller",
)

var allowedOptions = flag.String(
	"allowedOptions",
	"username,password,auto_cache,version",
	"A comma separated list of parameters allowed to be set in config.",
)

var defaultOptions = flag.String(
	"defaultOptions",
	"",
	"A comma separated list of defaults specified as param:value. If a parameter has a default value and is not in the allowed list, this default value becomes a fixed value that cannot be overridden",
)

var credhubURL = flag.String(
	"credhubURL",
	"",
	"(optional) CredHub server URL when using CredHub to store broker state",
)

var credhubCACertPath = flag.String(
	"credhubCACertPath",
	"",
	"(optional) Path to CA Cert for CredHub",
)

var uaaClientID = flag.String(
	"uaaClientID",
	"",
	"(optional) UAA client ID when using CredHub to store broker state",
)

var uaaClientSecret = flag.String(
	"uaaClientSecret",
	"",
	"(optional) UAA client secret when using CredHub to store broker state",
)

var uaaCACertPath = flag.String(
	"uaaCACertPath",
	"",
	"(optional) Path to CA Cert for UAA used for CredHub authorization",
)

var storeID = flag.String(
	"storeID",
	"smbbroker",
	"(optional) Store ID used to namespace instance details and bindings (credhub only)",
)

var (
	username string
	password string
)

func main() {
	parseCommandLine()
	parseEnvironment()

	checkParams()

	sink, err := lager.NewRedactingWriterSink(os.Stdout, lager.DEBUG, nil, nil)
	if err != nil {
		panic(err)
	}

	logger, logSink := lagerflags.NewFromSink("smbbroker", sink)
	logger.Info("starting")
	defer logger.Info("ends")

	server := createServer(logger)

	if dbgAddr := debugserver.DebugAddress(flag.CommandLine); dbgAddr != "" {
		server = utils.ProcessRunnerFor(grouper.Members{
			{"debug-server", debugserver.Runner(dbgAddr, logSink)},
			{"broker-api", server},
		})
	}

	process := ifrit.Invoke(server)
	logger.Info("started")
	utils.UntilTerminated(logger, process)
}

func parseCommandLine() {
	lagerflags.AddFlags(flag.CommandLine)
	debugserver.AddFlags(flag.CommandLine)
	flag.Parse()
}

func parseEnvironment() {
	username, _ = os.LookupEnv("USERNAME")
	password, _ = os.LookupEnv("PASSWORD")
}

func checkParams() {
	if *credhubURL == "" {
		fmt.Fprint(os.Stderr, "\nERROR: CredhubURL parameter must be provided.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *servicesConfig == "" {
		fmt.Fprint(os.Stderr, "\nERROR: servicesConfig parameter must be provided.\n\n")
		flag.Usage()
		os.Exit(1)
	}
}

func getByAlias(data map[string]interface{}, keys ...string) interface{} {
	for _, key := range keys {
		value, ok := data[key]
		if ok {
			return value
		}
	}
	return nil
}

func createServer(logger lager.Logger) ifrit.Runner {
	var credhubCACert string
	if *credhubCACertPath != "" {
		b, err := ioutil.ReadFile(*credhubCACertPath)
		if err != nil {
			logger.Fatal("cannot-read-credhub-ca-cert", err, lager.Data{"path": *credhubCACertPath})
		}
		credhubCACert = string(b)
	}

	var uaaCACert string
	if *uaaCACertPath != "" {
		b, err := ioutil.ReadFile(*uaaCACertPath)
		if err != nil {
			logger.Fatal("cannot-read-credhub-ca-cert", err, lager.Data{"path": *uaaCACertPath})
		}
		uaaCACert = string(b)
	}

	store := brokerstore.NewStore(
		logger,
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		*credhubURL,
		credhubCACert,
		*uaaClientID,
		*uaaClientSecret,
		uaaCACert,
		"",
		*storeID,
	)

	mounts := existingvolumebroker.NewExistingVolumeBrokerConfigDetails()
	mounts.ReadConf(*allowedOptions, *defaultOptions)
	logger.Debug("smbbroker-startup-config", lager.Data{"config": mounts})

	config := existingvolumebroker.NewExistingVolumeBrokerConfig(mounts)

	services, err := NewServicesFromConfig(*servicesConfig)
	if err != nil {
		logger.Fatal("loading-services-config-error", err)
	}

	serviceBroker := existingvolumebroker.New(
		existingvolumebroker.BrokerTypeSMB,
		logger,
		services,
		&osshim.OsShim{},
		clock.NewClock(),
		store,
		config,
	)

	credentials := brokerapi.BrokerCredentials{Username: username, Password: password}
	handler := brokerapi.New(serviceBroker, logger.Session("broker-api"), credentials)

	return http_server.New(*atAddress, handler)
}
