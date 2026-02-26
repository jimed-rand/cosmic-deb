package thermal

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	TempThresholdCritical = 90
	TempThresholdHigh     = 80
	TempThresholdWarn     = 70
	CooldownMin           = 15 * time.Minute
	CooldownMax           = 45 * time.Minute
	LowEndCoreThreshold   = 2
	LowEndThreadThreshold = 2
)

type Profile struct {
	IsLowEnd          bool
	PhysicalCores     int
	LogicalThreads    int
	MaxConcurrentJobs int
	CooldownDuration  time.Duration
}

type ThermalState struct {
	TempCelsius float64
	Available   bool
}

func ReadCPUTemp() ThermalState {
	paths := []string{
		"/sys/class/thermal/thermal_zone0/temp",
		"/sys/class/thermal/thermal_zone1/temp",
		"/sys/class/hwmon/hwmon0/temp1_input",
		"/sys/class/hwmon/hwmon1/temp1_input",
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		raw := strings.TrimSpace(string(data))
		val, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			continue
		}
		if val > 1000 {
			val /= 1000.0
		}
		return ThermalState{TempCelsius: val, Available: true}
	}
	return ThermalState{}
}

func ReadCPUTempFromHwmon() ThermalState {
	hwmonBase := "/sys/class/hwmon"
	entries, err := os.ReadDir(hwmonBase)
	if err != nil {
		return ThermalState{}
	}
	for _, e := range entries {
		dir := filepath.Join(hwmonBase, e.Name())
		nameFile := filepath.Join(dir, "name")
		nameBytes, err := os.ReadFile(nameFile)
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(nameBytes))
		if name != "coretemp" && name != "k10temp" && name != "zenpower" && name != "acpitz" {
			continue
		}
		inputFiles, err := filepath.Glob(filepath.Join(dir, "temp*_input"))
		if err != nil || len(inputFiles) == 0 {
			continue
		}
		var maxTemp float64
		for _, f := range inputFiles {
			data, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			val, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
			if err != nil {
				continue
			}
			if val > 1000 {
				val /= 1000.0
			}
			if val > maxTemp {
				maxTemp = val
			}
		}
		if maxTemp > 0 {
			return ThermalState{TempCelsius: maxTemp, Available: true}
		}
	}
	return ReadCPUTemp()
}

func DetectProfile() Profile {
	logical := runtime.NumCPU()
	physical := detectPhysicalCores()
	if physical <= 0 {
		physical = logical
	}

	isLowEnd := physical <= LowEndCoreThreshold && logical <= LowEndThreadThreshold

	maxJobs := logical
	cooldown := time.Duration(0)

	if isLowEnd {
		maxJobs = 1
		cooldown = CooldownMin
	}

	return Profile{
		IsLowEnd:          isLowEnd,
		PhysicalCores:     physical,
		LogicalThreads:    logical,
		MaxConcurrentJobs: maxJobs,
		CooldownDuration:  cooldown,
	}
}

func detectPhysicalCores() int {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return 0
	}
	seen := make(map[string]bool)
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "core id") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				seen[strings.TrimSpace(parts[1])] = true
			}
		}
	}
	if len(seen) > 0 {
		return len(seen)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "cpu cores") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				n, err := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err == nil && n > 0 {
					return n
				}
			}
		}
	}
	return 0
}

func ComputeCooldown(temp float64) time.Duration {
	if temp >= float64(TempThresholdCritical) {
		return CooldownMax
	}
	if temp >= float64(TempThresholdHigh) {
		ratio := (temp - float64(TempThresholdHigh)) / float64(TempThresholdCritical-TempThresholdHigh)
		span := CooldownMax - CooldownMin
		return CooldownMin + time.Duration(float64(span)*ratio)
	}
	if temp >= float64(TempThresholdWarn) {
		return CooldownMin
	}
	return 0
}

func WaitForCooldown(profile Profile, builtCount int, logFn func(string, ...any)) {
	if !profile.IsLowEnd {
		return
	}

	if builtCount == 0 || builtCount%2 != 0 {
		return
	}

	state := ReadCPUTempFromHwmon()

	var cooldown time.Duration
	if state.Available {
		cooldown = ComputeCooldown(state.TempCelsius)
		if cooldown == 0 {
			logFn("[Thermal] CPU temp %.1f°C is within safe range; no cooldown needed", state.TempCelsius)
			return
		}
		logFn("[Thermal] Low-end CPU detected. Temp: %.1f°C — Cooldown: %s", state.TempCelsius, formatDuration(cooldown))
	} else {
		cooldown = profile.CooldownDuration
		logFn("[Thermal] CPU temp sensor not readable. Applying default cooldown: %s", formatDuration(cooldown))
	}

	deadline := time.Now().Add(cooldown)
	logFn("[Thermal] Cooldown started. Build will resume at %s", deadline.Format("15:04:05"))

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for remaining := cooldown; remaining > 0; {
		select {
		case <-ticker.C:
			remaining = time.Until(deadline)
			currentState := ReadCPUTempFromHwmon()
			if currentState.Available {
				logFn("[Thermal] Cooldown in progress — %.0fs remaining. Current temp: %.1f°C",
					remaining.Seconds(), currentState.TempCelsius)
				if currentState.TempCelsius < float64(TempThresholdWarn) && remaining > 5*time.Minute {
					logFn("[Thermal] Temp dropped to %.1f°C. Reducing cooldown.", currentState.TempCelsius)
					deadline = time.Now().Add(5 * time.Minute)
				}
			} else {
				logFn("[Thermal] Cooldown in progress — %.0fs remaining", remaining.Seconds())
			}
			if remaining <= 0 {
				goto done
			}
		case <-time.After(remaining):
			goto done
		}
	}

done:
	logFn("[Thermal] Cooldown complete. Resuming build.")
}

func formatDuration(d time.Duration) string {
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	if s == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dm%ds", m, s)
}

func AdjustJobs(profile Profile, requestedJobs int) int {
	if !profile.IsLowEnd {
		return requestedJobs
	}
	if requestedJobs > profile.MaxConcurrentJobs {
		return profile.MaxConcurrentJobs
	}
	return requestedJobs
}

func SummarizeThermalProfile(profile Profile, logFn func(string, ...any)) {
	if profile.IsLowEnd {
		logFn("[Thermal] Low-end CPU profile active: %d physical core(s), %d thread(s)",
			profile.PhysicalCores, profile.LogicalThreads)
		logFn("[Thermal] Build limiter enabled: max %d parallel job(s), cooldown after every 2 components",
			profile.MaxConcurrentJobs)
	}
}
