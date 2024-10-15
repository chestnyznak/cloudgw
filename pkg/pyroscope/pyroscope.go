package pyroscope

import (
	"runtime"

	"github.com/grafana/pyroscope-go"
)

func EnablePyroscopeProfiling(appName, hostName, pyroscopeServerURL string) error {
	runtime.SetMutexProfileFraction(5)
	runtime.SetBlockProfileRate(5)

	if _, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: appName,
		ServerAddress:   pyroscopeServerURL,
		Logger:          nil,
		Tags:            map[string]string{"hostname": hostName},
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,

			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	}); err != nil {
		return err
	}

	return nil
}
