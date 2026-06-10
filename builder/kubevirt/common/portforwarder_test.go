// Copyright (c) Red Hat, Inc.
// SPDX-License-Identifier: MPL-2.0

package common_test

import (
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flippyboy/packer-plugin-kubevirt/builder/kubevirt/common"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
)

type mockStream struct {
	conn net.Conn
}

func (m *mockStream) Stream(_ kvcorev1.StreamOptions) error {
	return nil
}

func (m *mockStream) AsConn() net.Conn {
	return m.conn
}

type mockPortforwardResource struct {
	failures atomic.Int32
	attempts atomic.Int32
}

func (m *mockPortforwardResource) PortForward(_ string, _ int, _ string) (kvcorev1.StreamInterface, error) {
	attempt := m.attempts.Add(1)
	if int(attempt) <= int(m.failures.Load()) {
		return nil, fmt.Errorf("dial tcp: connect: no route to host")
	}

	client, server := net.Pipe()
	_ = client.Close()
	return &mockStream{conn: server}, nil
}

func TestPortForwarderSurvivesAPIError(t *testing.T) {
	resource := &mockPortforwardResource{}
	resource.failures.Store(1)

	forwarder := &common.PortForwarder{
		Kind:      "vm",
		Namespace: "test-ns",
		Name:      "test-vm",
		Resource:  resource,
	}

	address, err := net.ResolveIPAddr("ip4", "127.0.0.1")
	if err != nil {
		t.Fatalf("resolve address: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	localPort := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("close temp listener: %v", err)
	}

	if err := forwarder.StartForwarding(address, common.ForwardedPort{
		Local:    localPort,
		Remote:   5985,
		Protocol: common.ProtocolTCP,
	}); err != nil {
		t.Fatalf("start forwarding: %v", err)
	}
	t.Cleanup(func() {
		if err := forwarder.Close(); err != nil {
			t.Fatalf("close forwarder: %v", err)
		}
	})

	target := net.JoinHostPort("127.0.0.1", fmt.Sprint(localPort))

	first, err := net.DialTimeout("tcp", target, time.Second)
	if err != nil {
		t.Fatalf("first dial: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close first connection: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	second, err := net.DialTimeout("tcp", target, time.Second)
	if err != nil {
		t.Fatalf("second dial: %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatalf("close second connection: %v", err)
	}

	deadline := time.Now().Add(time.Second)
	for resource.attempts.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if got := resource.attempts.Load(); got != 2 {
		t.Fatalf("expected 2 port-forward attempts, got %d", got)
	}
}