package main

import (
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"encoding/json"
	"io/ioutil"

	"fmt"

	"os"
	"time"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	"github.com/pivotal-cf/brokerapi"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("smbbroker Main", func() {
	Context("Missing required args", func() {
		var process ifrit.Process

		It("shows usage when credhubURL is not provided", func() {
			var args []string

			volmanRunner := failRunner{
				Name:       "smbbroker",
				Command:    exec.Command(binaryPath, args...),
				StartCheck: "CredhubURL parameter must be provided.",
			}

			process = ifrit.Invoke(volmanRunner)
		})

		It("shows usage when servicesConfig is not provided", func() {
			args := []string{"-credhubURL", "credhub-url"}

			volmanRunner := failRunner{
				Name:       "smbbroker",
				Command:    exec.Command(binaryPath, args...),
				StartCheck: "servicesConfig parameter must be provided.",
			}

			process = ifrit.Invoke(volmanRunner)
		})

		AfterEach(func() {
			ginkgomon.Kill(process) // this is only if incorrect implementation leaves process running
		})
	})

	Context("Has required args", func() {
		var (
			args               []string
			listenAddr         string
			username, password string

			process ifrit.Process

			credhubServer *ghttp.Server
		)

		BeforeEach(func() {
			listenAddr = "0.0.0.0:" + strconv.Itoa(8999+GinkgoParallelNode())
			username = "admin"
			password = "password"

			os.Setenv("USERNAME", username)
			os.Setenv("PASSWORD", password)

			credhubServer = ghttp.NewServer()

			infoResponse := credhubInfoResponse{
				AuthServer: credhubInfoResponseAuthServer{
					URL: "some-auth-server-url",
				},
			}

			credhubServer.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/info"),
				ghttp.RespondWithJSONEncoded(http.StatusOK, infoResponse),
			))

			args = append(args, "-listenAddr", listenAddr)
			args = append(args, "-credhubURL", credhubServer.URL())
			args = append(args, "-servicesConfig", "./default_services.json")
		})

		JustBeforeEach(func() {
			volmanRunner := ginkgomon.New(ginkgomon.Config{
				Name:       "smbbroker",
				Command:    exec.Command(binaryPath, args...),
				StartCheck: "started",
			})

			process = ginkgomon.Invoke(volmanRunner)
		})

		AfterEach(func() {
			ginkgomon.Kill(process)

			credhubServer.Close()
		})

		httpDoWithAuth := func(method, endpoint string, body io.Reader) (*http.Response, error) {
			req, err := http.NewRequest(method, "http://"+listenAddr+endpoint, body)
			req.Header.Add("X-Broker-Api-Version", "2.14")
			Expect(err).NotTo(HaveOccurred())

			req.SetBasicAuth(username, password)
			return http.DefaultClient.Do(req)
		}

		It("should listen on the given address", func() {
			resp, err := httpDoWithAuth("GET", "/v2/catalog", nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.StatusCode).To(Equal(200))
		})

		It("should pass services config through to catalog", func() {
			resp, err := httpDoWithAuth("GET", "/v2/catalog", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			bytes, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			var catalog brokerapi.CatalogResponse
			err = json.Unmarshal(bytes, &catalog)
			Expect(err).NotTo(HaveOccurred())

			Expect(catalog.Services[0].ID).To(Equal("9db9cca4-8fd5-4b96-a4c7-0a48f47c3bad"))
			Expect(catalog.Services[0].Name).To(Equal("smb"))
			Expect(catalog.Services[0].Plans[0].ID).To(Equal("0da18102-48dc-46d0-98b3-7a4ff6dc9c54"))
			Expect(catalog.Services[0].Plans[0].Name).To(Equal("Existing"))
			Expect(catalog.Services[0].Plans[0].Description).To(Equal("A preexisting share"))
		})

		Context("#update", func() {
			It("should respond with a 422", func() {
				updateDetailsJson, err := json.Marshal(brokerapi.UpdateDetails{
					ServiceID: "service-id",
				})
				Expect(err).NotTo(HaveOccurred())
				reader := strings.NewReader(string(updateDetailsJson))
				resp, err := httpDoWithAuth("PATCH", "/v2/service_instances/12345", reader)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(422))

				responseBody, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(responseBody)).To(ContainSubstring("This service does not support instance updates. Please delete your service instance and create a new one with updated configuration."))
			})
		})
	})
})

func (r failRunner) Run(sigChan <-chan os.Signal, ready chan<- struct{}) error {
	defer GinkgoRecover()

	allOutput := gbytes.NewBuffer()

	debugWriter := gexec.NewPrefixedWriter(
		fmt.Sprintf("\x1b[32m[d]\x1b[%s[%s]\x1b[0m ", r.AnsiColorCode, r.Name),
		GinkgoWriter,
	)

	session, err := gexec.Start(
		r.Command,
		gexec.NewPrefixedWriter(
			fmt.Sprintf("\x1b[32m[o]\x1b[%s[%s]\x1b[0m ", r.AnsiColorCode, r.Name),
			io.MultiWriter(allOutput, GinkgoWriter),
		),
		gexec.NewPrefixedWriter(
			fmt.Sprintf("\x1b[91m[e]\x1b[%s[%s]\x1b[0m ", r.AnsiColorCode, r.Name),
			io.MultiWriter(allOutput, GinkgoWriter),
		),
	)

	Expect(err).ShouldNot(HaveOccurred())

	fmt.Fprintf(debugWriter, "spawned %s (pid: %d)\n", r.Command.Path, r.Command.Process.Pid)

	r.session = session
	if r.sessionReady != nil {
		close(r.sessionReady)
	}

	startCheckDuration := r.StartCheckTimeout
	if startCheckDuration == 0 {
		startCheckDuration = 5 * time.Second
	}

	var startCheckTimeout <-chan time.Time
	if r.StartCheck != "" {
		startCheckTimeout = time.After(startCheckDuration)
	}

	detectStartCheck := allOutput.Detect(r.StartCheck)

	for {
		select {
		case <-detectStartCheck: // works even with empty string
			allOutput.CancelDetects()
			startCheckTimeout = nil
			detectStartCheck = nil
			close(ready)

		case <-startCheckTimeout:
			// clean up hanging process
			session.Kill().Wait()

			// fail to start
			return fmt.Errorf(
				"did not see %s in command's output within %s. full output:\n\n%s",
				r.StartCheck,
				startCheckDuration,
				string(allOutput.Contents()),
			)

		case signal := <-sigChan:
			session.Signal(signal)

		case <-session.Exited:
			if r.Cleanup != nil {
				r.Cleanup()
			}

			Expect(string(allOutput.Contents())).To(ContainSubstring(r.StartCheck))
			Expect(session.ExitCode()).To(Not(Equal(0)), fmt.Sprintf("Expected process to exit with non-zero, got: 0"))
			return nil
		}
	}
}

type credhubInfoResponse struct {
	AuthServer credhubInfoResponseAuthServer `json:"auth-server"`
}

type credhubInfoResponseAuthServer struct {
	URL string `json:"url"`
}

type failRunner struct {
	Command           *exec.Cmd
	Name              string
	AnsiColorCode     string
	StartCheck        string
	StartCheckTimeout time.Duration
	Cleanup           func()
	session           *gexec.Session
	sessionReady      chan struct{}
}
