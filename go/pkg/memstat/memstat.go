package memstat

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

// Print Go memory and garbage collector stats. Useful for debugging
func PrintMemoryStats() {
	mem := MemStats()

	fmt.Printf("\u001b[33m---- Memory Dump ----\u001b[39m\n")
	fmt.Printf("Allocated: %s\n", formatBytes(mem.Alloc))
	fmt.Printf("Total Allocated: %s\n", formatBytes(mem.TotalAlloc))
	fmt.Printf("Memory Allocations: %d\n", mem.Mallocs)
	fmt.Printf("Memory Frees: %d\n", mem.Frees)
	fmt.Printf("Heap Allocated: %s\n", formatBytes(mem.HeapAlloc))
	fmt.Printf("Heap System: %s\n", formatBytes(mem.HeapSys))
	fmt.Printf("Heap In Use: %s\n", formatBytes(mem.HeapInuse))
	fmt.Printf("Heap Idle: %s\n", formatBytes(mem.HeapIdle))
	fmt.Printf("Heap OS Related: %s\n", formatBytes(mem.HeapReleased))
	fmt.Printf("Heap Objects: %s\n", formatBytes(mem.HeapObjects))
	fmt.Printf("Stack In Use: %s\n", formatBytes(mem.StackInuse))
	fmt.Printf("Stack System: %s\n", formatBytes(mem.StackSys))
	fmt.Printf("Stack Span In Use: %s\n", formatBytes(mem.MSpanInuse))
	fmt.Printf("Stack Cache In Use: %s\n", formatBytes(mem.MCacheInuse))
	fmt.Printf("Next GC cycle: %s\n", formatNano(mem.NextGC))
	fmt.Printf("Last GC cycle: %s\n", formatNano(mem.LastGC))
	fmt.Printf("\u001b[33m---- Memory Stats ----\u001b[39m\n")
}

// Get the memory stats
func MemStats() runtime.MemStats {
	runtime.GC()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	return mem
}

// formatBytes converts bytes to human readable string. Like 2 MiB, 64.2 KiB, 52 B
func formatBytes(i uint64) (result string) {
	switch {
	case i > (1024 * 1024 * 1024 * 1024):
		result = fmt.Sprintf("%.02f TiB", float64(i)/1024/1024/1024/1024)
	case i > (1024 * 1024 * 1024):
		result = fmt.Sprintf("%.02f GiB", float64(i)/1024/1024/1024)
	case i > (1024 * 1024):
		result = fmt.Sprintf("%.02f MiB", float64(i)/1024/1024)
	case i > 1024:
		result = fmt.Sprintf("%.02f KiB", float64(i)/1024)
	default:
		result = fmt.Sprintf("%d B", i)
	}
	result = strings.Trim(result, " ")
	return
}

func formatNano(n uint64) string {
	var suffix string

	switch {
	case n > 1e9:
		n /= 1e9
		suffix = "s"
	case n > 1e6:
		n /= 1e6
		suffix = "ms"
	case n > 1e3:
		n /= 1e3
		suffix = "us"
	default:
		suffix = "ns"
	}

	return strconv.Itoa(int(n)) + suffix
}
