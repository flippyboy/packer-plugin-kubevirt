// Copyright (c) Red Hat, Inc.
// SPDX-License-Identifier: MPL-2.0

package iso_test

import (
	"context"
	"io"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flippyboy/packer-plugin-kubevirt/builder/kubevirt/iso"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

var _ = Describe("StepWaitForInstallation", func() {
	var (
		state *multistep.BasicStateBag
		step  *iso.StepWaitForInstallation
	)

	BeforeEach(func() {
		errorBuffer := &strings.Builder{}
		ui := &packer.BasicUi{
			Reader:      strings.NewReader(""),
			Writer:      io.Discard,
			ErrorWriter: errorBuffer,
		}
		state = new(multistep.BasicStateBag)
		state.Put("ui", ui)
	})

	Context("Run", func() {
		It("waits for the specified duration and continues", func() {
			step = &iso.StepWaitForInstallation{
				Config: iso.Config{InstallationWaitTimeout: 2 * time.Second},
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			action := step.Run(ctx, state)
			Expect(action).To(Equal(multistep.ActionContinue))
		})

		It("continues immediately when no wait time specified", func() {
			step = &iso.StepWaitForInstallation{
				Config: iso.Config{InstallationWaitTimeout: 0},
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			action := step.Run(ctx, state)
			Expect(action).To(Equal(multistep.ActionContinue))
		})

		It("halts when context is cancelled before wait time elapses", func() {
			step = &iso.StepWaitForInstallation{
				Config: iso.Config{InstallationWaitTimeout: 5 * time.Second},
			}

			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				time.Sleep(1 * time.Second)
				cancel()
			}()

			action := step.Run(ctx, state)
			Expect(action).To(Equal(multistep.ActionHalt))
		})
	})
})
