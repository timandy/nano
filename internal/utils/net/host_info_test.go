package net

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultIp(t *testing.T) {
	ip := net.IP(DefaultIpRaw).String()
	assert.Equal(t, DefaultIp, ip)
}

func TestDefaultMac(t *testing.T) {
	mac := hi.getPhysicalAddress(DefaultMacRaw)
	assert.Equal(t, DefaultMac, mac)
}

func TestHostName(t *testing.T) {
	assert.Greater(t, len(HostName()), 0)
}

func TestProcessId(t *testing.T) {
	assert.Greater(t, ProcessId(), 0)
}

func TestRawMacAddress(t *testing.T) {
	assert.Greater(t, len(RawMacAddress()), 0)
}

func TestMacAddress(t *testing.T) {
	assert.Equal(t, len(RawMacAddress()), len(MacAddress()))
}

func TestRawIpAddress(t *testing.T) {
	assert.Greater(t, len(RawIpAddress()), 0)
}

func TestIpAddress(t *testing.T) {
	assert.Equal(t, len(RawIpAddress()), len(IpAddress()))
}
