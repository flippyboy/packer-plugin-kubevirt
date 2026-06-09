// Copyright (c) Red Hat, Inc.
// SPDX-License-Identifier: MPL-2.0

package iso_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flippyboy/packer-plugin-kubevirt/builder/kubevirt/iso"
)

var _ = Describe("buildConfigMapData", func() {
	It("uses media_files for Linux", func() {
		err := os.WriteFile("ks.cfg", []byte("kickstart"), 0644)
		Expect(err).NotTo(HaveOccurred())
		defer os.Remove("ks.cfg")

		data, err := iso.BuildConfigMapData(iso.Config{
			OperatingSystemType: "linux",
			MediaFiles:          []string{"ks.cfg"},
			SysprepContent: map[string]string{
				"autounattend.xml": "ignored",
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(data).To(HaveKeyWithValue("ks.cfg", "kickstart"))
		Expect(data).NotTo(HaveKey("autounattend.xml"))
	})

	It("merges sysprep_files, media_files, and sysprep_content for Windows", func() {
		for _, name := range []string{"autounattend.xml", "install.ps1", "legacy.ps1"} {
			err := os.WriteFile(name, []byte("file:"+name), 0644)
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(name)
		}

		data, err := iso.BuildConfigMapData(iso.Config{
			OperatingSystemType: "windows",
			SysprepFiles:        []string{"install.ps1"},
			MediaFiles:          []string{"legacy.ps1"},
			SysprepContent: map[string]string{
				"autounattend.xml": "templated content",
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(data).To(HaveKeyWithValue("install.ps1", "file:install.ps1"))
		Expect(data).To(HaveKeyWithValue("legacy.ps1", "file:legacy.ps1"))
		Expect(data).To(HaveKeyWithValue("autounattend.xml", "templated content"))
	})
})