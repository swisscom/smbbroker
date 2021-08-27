package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi"

	. "code.cloudfoundry.org/smbbroker"
)

var _ = Describe("Services", func() {
	var (
		services Services
	)

	BeforeEach(func() {
		var err error
		services, err = NewServicesFromConfig("./default_services.json")
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("List", func() {
		It("returns the list of services", func() {
			Expect(services.List()).To(Equal([]brokerapi.Service{
				{
					ID:            "9db9cca4-8fd5-4b96-a4c7-0a48f47c3bad",
					Name:          "smb",
					Description:   "Existing SMB shares (see: https://code.cloudfoundry.org/smb-volume-release/)",
					Bindable:      true,
					PlanUpdatable: false,
					Tags:          []string{"smb"},
					Requires:      []brokerapi.RequiredPermission{"volume_mount"},

					Plans: []brokerapi.ServicePlan{
						{
							ID:          "0da18102-48dc-46d0-98b3-7a4ff6dc9c54",
							Name:        "existing",
							Description: "A preexisting share",
						},
					},
				},
			}))
		})
	})
})
