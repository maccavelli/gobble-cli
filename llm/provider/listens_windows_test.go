package provider

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// GetExtendedTcpTable's arguments and results this file uses (iphlpapi.h,
// tcpmib.h).
const (
	tcpTableOwnerPIDListener = 3 // TCP_TABLE_OWNER_PID_LISTENER
	afINET                   = 2
	afINET6                  = 23
	errInsufficientBuffer    = 122 // ERROR_INSUFFICIENT_BUFFER
)

var procGetExtendedTCPTable = syscall.NewLazyDLL("iphlpapi.dll").NewProc("GetExtendedTcpTable")

// tcpRowLayout is where a listener table's row holds the local port and the
// owning process: MIB_TCPROW_OWNER_PID (24 bytes, dwLocalPort at 8,
// dwOwningPid at 20) and MIB_TCP6ROW_OWNER_PID (56 bytes, at 20 and 52). The
// port is in network byte order in the DWORD's first two bytes.
type tcpRowLayout struct{ af, size, port, pid int }

var tcpRowLayouts = []tcpRowLayout{{afINET, 24, 8, 20}, {afINET6, 56, 20, 52}}

// ownListeningPorts reads the IPv4 and IPv6 listener tables and keeps the
// local port of each row this process owns.
func ownListeningPorts() (map[int]bool, error) {
	pid := uint32(os.Getpid()) // a Windows process id is a DWORD
	ports := map[int]bool{}
	for _, l := range tcpRowLayouts {
		table, err := listenerTable(l.af)
		if err != nil {
			return nil, err
		}
		if len(table) < 4 {
			return nil, fmt.Errorf("GetExtendedTcpTable(%d): a table of %d bytes", l.af, len(table))
		}
		n := int(binary.LittleEndian.Uint32(table))
		if 4+n*l.size > len(table) {
			return nil, fmt.Errorf("GetExtendedTcpTable(%d): %d rows in %d bytes", l.af, n, len(table))
		}
		for i := range n {
			row := table[4+i*l.size : 4+(i+1)*l.size]
			if binary.LittleEndian.Uint32(row[l.pid:]) == pid {
				ports[int(binary.BigEndian.Uint16(row[l.port:]))] = true
			}
		}
	}
	return ports, nil
}

// listenerTable is GetExtendedTcpTable's listener table for address family
// af. The first call asks the size; the table can grow before the next, which
// then asks again with the size it reports.
func listenerTable(af int) ([]byte, error) {
	var size uint32
	r, _, _ := procGetExtendedTCPTable.Call(0, uintptr(unsafe.Pointer(&size)), 0, uintptr(af), tcpTableOwnerPIDListener, 0)
	for range 8 {
		switch r {
		case 0:
			if size == 0 {
				return nil, errors.New("GetExtendedTcpTable: no size")
			}
		case errInsufficientBuffer:
		default:
			return nil, fmt.Errorf("GetExtendedTcpTable(%d): %w", af, syscall.Errno(r))
		}
		buf := make([]byte, size)
		r, _, _ = procGetExtendedTCPTable.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)), 0, uintptr(af), tcpTableOwnerPIDListener, 0)
		if r == 0 {
			return buf, nil
		}
	}
	return nil, fmt.Errorf("GetExtendedTcpTable(%d): the table kept growing", af)
}
