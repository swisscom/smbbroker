package main_test

import (
	. "code.cloudfoundry.org/smbbroker"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {

	It("should return the correct allowed options", func() {
		Expect(AllowedOptions()).To(Equal("source,mount,ro,username,password,domain,version,mfsymlinks"))
	})

})
