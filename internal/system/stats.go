package system

import (
	"fmt"

	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

// DashboardData holds the stats we send to the HTML
type DashboardData struct {
	CPU      string
	RAM      string
	Disk     string
	Uptime   string
	Hostname string
	OS       string
}

// GetStats gathers all system information
func GetStats() DashboardData {
	// 1. CPU Usage (Get average of last 1 second)
	c, _ := cpu.Percent(time.Second, false)
	cpuVal := 0.0
	if len(c) > 0 {
		cpuVal = c[0]
	}

	// 2. Memory Usage
	v, _ := mem.VirtualMemory()

	// 3. Disk Usage (Root partition)
	d, _ := disk.Usage("/")

	// 4. Host Info (Uptime, OS)
	h, _ := host.Info()

	// Format Uptime (Seconds -> Days/Hours)
	uptimeDur := time.Duration(h.Uptime) * time.Second
	days := int(uptimeDur.Hours()) / 24
	hours := int(uptimeDur.Hours()) % 24

	return DashboardData{
		CPU:      fmt.Sprintf("%.1f%%", cpuVal),
		RAM:      fmt.Sprintf("%.1f GB / %.1f GB", float64(v.Used)/1024/1024/1024, float64(v.Total)/1024/1024/1024),
		Disk:     fmt.Sprintf("%.0f%%", d.UsedPercent),
		Uptime:   fmt.Sprintf("%dd %dh", days, hours),
		Hostname: h.Hostname,
		OS:       h.Platform + " " + h.PlatformVersion,
	}
}
