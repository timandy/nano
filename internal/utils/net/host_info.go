package net

import (
	"net"
	"os"
)

const (
	DefaultIp  = "127.0.0.1"
	DefaultMac = "75-6E-6B-6E-6F-77"
)

var (
	DefaultIpRaw  = []byte{127, 0, 0, 1}
	DefaultMacRaw = []byte{'u', 'n', 'k', 'n', 'o', 'w'}
)

type hostInfo struct {
	hostName      string
	processId     int
	rawMacAddress [][]byte
	rawIpAddress  [][]byte
	macAddress    []string
	ipAddress     []string
}

func (h *hostInfo) getHexValue(i int32) rune {
	if i < 10 {
		return i + '0'
	}
	return i - 10 + 'A'
}

func (h *hostInfo) getPhysicalAddress(mac []byte) string {
	arrayLen := len(mac) * 3
	array := make([]rune, arrayLen)
	for byteIndex, charIndex := 0, 0; charIndex < arrayLen; {
		b := mac[byteIndex]
		array[charIndex] = h.getHexValue((int32(b) & 0xF0) >> 4)
		array[charIndex+1] = h.getHexValue(int32(b) & 0x0F)
		array[charIndex+2] = '-'
		byteIndex++
		charIndex += 3
	}
	return string(array[:arrayLen-1])
}

func (h *hostInfo) refresh() {
	//主机名
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	h.hostName = hostname
	//进程id
	h.processId = os.Getpid()
	//ip 信息
	interfaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	for _, inter := range interfaces {
		if inter.Flags&net.FlagUp == 0 ||
			inter.Flags&net.FlagLoopback != 0 ||
			inter.Flags&net.FlagPointToPoint != 0 {
			continue
		}
		mac := []byte(inter.HardwareAddr)
		if len(mac) == 0 {
			continue
		}
		h.rawMacAddress = append(h.rawMacAddress, mac)
		h.macAddress = append(h.macAddress, h.getPhysicalAddress(mac))
		addrs, err2 := inter.Addrs()
		if err2 != nil {
			continue
		}
		for _, addr := range addrs {
			ipn, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipn.IP
			if ip.IsLoopback() || ip.IsMulticast() {
				continue
			}
			if len(ip) == 0 {
				continue
			}
			ip4 := ip.To4()
			if ip4 == nil {
				continue
			}
			h.rawIpAddress = append(h.rawIpAddress, ip4)
			h.ipAddress = append(h.ipAddress, ip4.String())
		}
	}
}

// 对外接口
var hi *hostInfo

func init() {
	h := &hostInfo{}
	h.refresh()
	hi = h
}

func HostName() string {
	return hi.hostName
}

func ProcessId() int {
	return hi.processId
}

func RawMacAddress() [][]byte {
	raw := hi.rawMacAddress
	if len(raw) == 0 {
		return [][]byte{DefaultMacRaw}
	}
	return raw
}

func RawIpAddress() [][]byte {
	raw := hi.rawIpAddress
	if len(raw) == 0 {
		return [][]byte{DefaultIpRaw}
	}
	return raw
}

func MacAddress() []string {
	v := hi.macAddress
	if len(v) == 0 {
		return []string{DefaultMac}
	}
	return v
}

func IpAddress() []string {
	v := hi.ipAddress
	if len(v) == 0 {
		return []string{DefaultIp}
	}
	return v
}
