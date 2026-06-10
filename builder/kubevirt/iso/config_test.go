// Copyright (c) Red Hat, Inc.
// SPDX-License-Identifier: MPL-2.0

package iso_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flippyboy/packer-plugin-kubevirt/builder/kubevirt/iso"
)

var _ = Describe("Config Prepare", func() {
	It("rejects sysprep options when os_type is not windows", func() {
		cfg := iso.Config{
			OperatingSystemType: "linux",
			SysprepContent: map[string]string{
				"autounattend.xml": "<unattend/>",
			},
		}

		_, err := cfg.Prepare(nil)
		Expect(err).To(MatchError(`sysprep_files and sysprep_content are only supported when os_type is "windows"`))
	})
})
