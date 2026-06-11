/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package common

import (
	"errors"
	"io"
	"net"
	"strings"
	"sync"

	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
	"kubevirt.io/client-go/log"
)

const (
	ProtocolTCP = "tcp"
)

type PortForward struct {
	Address  *net.IPAddr
	Resource PortforwardableResource
}

type PortForwarder struct {
	Kind, Namespace, Name string
	Resource              PortforwardableResource

	mu       sync.Mutex
	listener net.Listener
}

type ForwardedPort struct {
	Local    int
	Remote   int
	Protocol string
}

type PortforwardableResource interface {
	PortForward(name string, port int, protocol string) (kvcorev1.StreamInterface, error)
}

func (p *PortForwarder) StartForwarding(address *net.IPAddr, port ForwardedPort) (int, error) {
	if port.Protocol == ProtocolTCP {
		return p.StartForwardingTCP(address, port)
	}
	return 0, errors.New("unknown protocol: " + port.Protocol)
}

func (p *PortForwarder) StartForwardingTCP(address *net.IPAddr, port ForwardedPort) (int, error) {
	listener, err := net.ListenTCP(
		port.Protocol,
		&net.TCPAddr{
			IP:   address.IP,
			Zone: address.Zone,
			Port: port.Local,
		})
	if err != nil {
		return 0, err
	}

	boundPort := listener.Addr().(*net.TCPAddr).Port
	port.Local = boundPort
	log.Log.Infof("forwarding %s %s:%d to %d", port.Protocol, address, boundPort, port.Remote)

	p.mu.Lock()
	p.listener = listener
	p.mu.Unlock()

	go p.WaitForConnection(listener, port)
	return boundPort, nil
}

func (p *PortForwarder) Close() error {
	p.mu.Lock()
	listener := p.listener
	p.listener = nil
	p.mu.Unlock()

	if listener != nil {
		return listener.Close()
	}
	return nil
}

func (p *PortForwarder) WaitForConnection(listener net.Listener, port ForwardedPort) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			if isListenerClosed(err) {
				log.Log.Infof("stopped forwarding local port %d", port.Local)
				return
			}
			log.Log.Errorf("error accepting connection: %v", err)
			return
		}
		log.Log.Infof("opening new tcp tunnel to %d", port.Remote)
		stream, err := p.Resource.PortForward(p.Name, port.Remote, port.Protocol)
		if err != nil {
			log.Log.Warningf("can't access %s/%s.%s: %v", p.Kind, p.Name, p.Namespace, err)
			conn.Close()
			continue
		}
		go p.HandleConnection(conn, stream.AsConn(), port)
	}
}

// handleConnection copies data between the local connection and the stream to
// the remote server.
func (p *PortForwarder) HandleConnection(local, remote net.Conn, port ForwardedPort) {
	log.Log.Infof("handling tcp connection for %d", port.Local)
	errs := make(chan error)
	go func() {
		_, err := io.Copy(remote, local)
		errs <- err
	}()
	go func() {
		_, err := io.Copy(local, remote)
		errs <- err
	}()

	HandleConnectionError(<-errs, port)
	local.Close()
	remote.Close()
	HandleConnectionError(<-errs, port)
}

func HandleConnectionError(err error, port ForwardedPort) {
	if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
		log.Log.Errorf("error handling connection for %d: %v", port.Local, err)
	}
}

func isListenerClosed(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, net.ErrClosed) {
		return true
	}
	return strings.Contains(err.Error(), "use of closed network connection")
}
