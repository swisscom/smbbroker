package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "code.cloudfoundry.org/smbbroker"
)

var _ = Describe("Config", func() {

	It("should return the correct allowed options", func() {
		Expect(AllowedOptions()).To(Equal("source,mount,ro,username,password,domain"))
	})

})
