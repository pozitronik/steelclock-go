package metrics

// NetworkStat represents network I/O statistics for an interface
type NetworkStat struct {
	Name      string // Interface name
	BytesRecv uint64 // Total bytes received
	BytesSent uint64 // Total bytes sent
}

// DiskStat represents disk I/O statistics for a device
type DiskStat struct {
	Name       string // Device name
	ReadBytes  uint64 // Total bytes read
	WriteBytes uint64 // Total bytes written
}
