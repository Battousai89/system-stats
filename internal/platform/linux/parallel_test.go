//go:build linux
// +build linux

package linux

import (
	"sync"
	"testing"
	"time"
)

// TestParallelExecution tests that all stat collection functions can run in parallel
func TestParallelExecution(t *testing.T) {
	var wg sync.WaitGroup
	errors := make(chan error, 7)
	results := make(chan string, 7)

	startTime := time.Now()

	// Run all stat collectors in parallel
	wg.Add(7)

	go func() {
		defer wg.Done()
		info, err := NewHostInfo()
		if err != nil {
			errors <- err
			return
		}
		results <- "host:" + info.Hostname
	}()

	go func() {
		defer wg.Done()
		info, err := NewCPUInfo()
		if err != nil {
			errors <- err
			return
		}
		results <- "cpu:" + string(rune(len(info)))
	}()

	go func() {
		defer wg.Done()
		mem, err := GetVirtualMemory()
		if err != nil {
			errors <- err
			return
		}
		results <- "mem:" + string(rune(mem.Total))
	}()

	go func() {
		defer wg.Done()
		usage, err := NewDiskUsage("/")
		if err != nil {
			errors <- err
			return
		}
		results <- "disk:" + string(rune(usage.Total))
	}()

	go func() {
		defer wg.Done()
		counters, err := NewNetIOCounters()
		if err != nil {
			errors <- err
			return
		}
		results <- "net:" + string(rune(len(counters)))
	}()

	go func() {
		defer wg.Done()
		processes, err := NewProcessInfo(5)
		if err != nil {
			errors <- err
			return
		}
		results <- "proc:" + string(rune(len(processes)))
	}()

	go func() {
		defer wg.Done()
		sensors, err := NewSensorTemperatures()
		if err != nil {
			errors <- err
			return
		}
		results <- "sensors:" + string(rune(len(sensors)))
	}()

	// Wait for all goroutines
	go func() {
		wg.Wait()
		close(results)
		close(errors)
	}()

	// Collect results
	for result := range results {
		t.Logf("Result: %s", result)
	}

	// Check for errors
	for err := range errors {
		t.Errorf("Parallel execution error: %v", err)
	}

	elapsed := time.Since(startTime)
	t.Logf("Parallel execution completed in %v", elapsed)

	// Parallel execution should be faster than sequential
	// If it takes more than 5 seconds, something is wrong
	if elapsed > 5*time.Second {
		t.Errorf("Parallel execution took too long: %v", elapsed)
	}
}

// TestConcurrentAccess tests that functions can be called concurrently
func TestConcurrentAccess(t *testing.T) {
	var wg sync.WaitGroup
	iterations := 10

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(iter int) {
			defer wg.Done()
			
			_, err := NewHostInfo()
			if err != nil {
				t.Errorf("Iteration %d: NewHostInfo() error: %v", iter, err)
			}
			
			_, err = GetVirtualMemory()
			if err != nil {
				t.Errorf("Iteration %d: GetVirtualMemory() error: %v", iter, err)
			}
		}(i)
	}

	wg.Wait()
}

// BenchmarkParallelCPUStats benchmarks parallel CPU stats collection
func BenchmarkParallelCPUStats(b *testing.B) {
	var wg sync.WaitGroup
	
	for i := 0; i < b.N; i++ {
		wg.Add(3)
		
		go func() {
			defer wg.Done()
			NewCPUInfo()
		}()
		
		go func() {
			defer wg.Done()
			NewCPUTimes()
		}()
		
		go func() {
			defer wg.Done()
			NewCPUPercent()
		}()
	}
	
	wg.Wait()
}

// BenchmarkParallelAllStats benchmarks parallel collection of all stats
func BenchmarkParallelAllStats(b *testing.B) {
	var wg sync.WaitGroup
	
	for i := 0; i < b.N; i++ {
		wg.Add(5)
		
		go func() {
			defer wg.Done()
			NewHostInfo()
		}()
		
		go func() {
			defer wg.Done()
			GetVirtualMemory()
		}()
		
		go func() {
			defer wg.Done()
			NewDiskUsage("/")
		}()
		
		go func() {
			defer wg.Done()
			NewNetIOCounters()
		}()
		
		go func() {
			defer wg.Done()
			NewProcessInfo(5)
		}()
	}
	
	wg.Wait()
}
