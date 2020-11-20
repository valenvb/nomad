package stats

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/shirou/gopsutil/cpu"
)

const (
	// cpuInfoTimeout is the timeout used when gathering CPU info. This is used
	// to override the default timeout in gopsutil which has a tendency to
	// timeout on Windows.
	cpuInfoTimeout = 60 * time.Second
)

var (
	cpuMhzPerCore float64
	cpuModelName  string
	cpuNumCores   int
	cpuTotalTicks float64

	initErr error
	onceLer sync.Once
)

func Init() error {
	onceLer.Do(func() {
		var merrs *multierror.Error
		var err error
		if cpuNumCores, err = cpu.Counts(true); err != nil {
			merrs = multierror.Append(merrs, fmt.Errorf("Unable to determine the number of CPU cores available: %v", err))
		}

		var infoStats []cpu.InfoStat
		ctx, cancel := context.WithTimeout(context.Background(), cpuInfoTimeout)
		defer cancel()
		if infoStats, err = cpu.InfoWithContext(ctx); err != nil {
			merrs = multierror.Append(merrs, fmt.Errorf("Unable to obtain CPU information: %v", err))
		}

		if len(infoStats) > 0 {
			cpuModelName = infoStats[0].ModelName
			cpuMhzPerCore = infoStats[0].Mhz
		}

		// Floor all of the values such that small difference don't cause the
		// node to fall into a unique computed node class
		cpuMhzPerCore = math.Floor(cpuMhzPerCore)
		cpuTotalTicks = math.Floor(float64(cpuNumCores) * cpuMhzPerCore)

		// A kludge for a growing number of systems where CPU frequency is not
		// available from the operating system, and must be acquired from other
		// means like EC2 API's and other lookup tables.
		if cpuTotalTicks <= 0 {
			cpuTotalTicks = 1 // another fingerprinter will update this
		}

		// Set any errors that occurred
		initErr = merrs.ErrorOrNil()
	})
	return initErr
}

// CPUModelName returns the number of CPU cores available
func CPUNumCores() int {
	return cpuNumCores
}

// CPUMHzPerCore returns the MHz per CPU core
func CPUMHzPerCore() float64 {
	return cpuMhzPerCore
}

// CPUModelName returns the model name of the CPU
func CPUModelName() string {
	return cpuModelName
}

// TotalTicksAvailable calculates the total Mhz available across all cores
func TotalTicksAvailable() float64 {
	return cpuTotalTicks
}
