package zpool

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

type ZPool struct {
	name     string
	capacity int64
	healthy  bool
	status   string
	online   int64
	faulted  int64
}

func (z *ZPool) checkHealth(output string) (err error) {
	output = strings.Trim(output, "\n")
	if output == "ONLINE" {
		z.healthy = true
	} else if output == "DEGRADED" || output == "FAULTED" {
		z.healthy = false
	} else {
		z.healthy = false // just to make sure
		err = errors.New("Unknown status")
	}
	return err
}

func (z *ZPool) getCapacity(output string) (err error) {
	s := strings.Split(output, "%")[0]
	z.capacity, err = strconv.ParseInt(s, 0, 8)
	if err != nil {
		return err
	}
	return err
}

func (z *ZPool) getProviders(output string) (err error) {
	nonProviderLines := []string{
		z.name,
		"state:",
		"mirror-",
		"raid0-",
		"raid10-",
		"raidz-",
		"raidz2-",
		"raidz3-",
	}
	lines := strings.Split(output, "\n")
	z.status = strings.Split(lines[1], " ")[2]

	// Count all providers, ONLINE and FAULTED
	var fcount int64
	var dcount int64
	for _, line := range lines {
		if (strings.Contains(line, "FAULTED") || strings.Contains(line, "UNAVAIL")) && !substringInSlice(line, nonProviderLines) {
			fcount = fcount + 1
		} else if strings.Contains(line, "ONLINE") && !substringInSlice(line, nonProviderLines) {
			dcount = dcount + 1
		}
	}
	z.faulted = fcount
	z.online = dcount

	if z.status != "ONLINE" && z.status != "DEGRADED" && z.status != "FAULTED" {
		z.faulted = 1 // fake faulted if there is a parsing error or other status
		err = errors.New("Error parsing faulted/unavailable providers")
	}
	return
}

func (z *ZPool) GetStatus() {
	output := RunZpoolCommand([]string{"status", z.name})
	err := z.getProviders(output)
	if err != nil {
		log.Fatal("Error parsing zpool status")
	}
	output = RunZpoolCommand([]string{"list", "-H", "-o", "health", z.name})
	err = z.checkHealth(output)
	if err != nil {
		log.Fatal("Error parsing zpool list -H -o health ", z.name)
	}
	output = RunZpoolCommand([]string{"list", "-H", "-o", "cap", z.name})
	err = z.getCapacity(output)
	if err != nil {
		log.Fatal("Error parsing zpool capacity")
	}
}

func CheckExistance(pool string) (err error) {
	output := RunZpoolCommand([]string{"list", pool})
	if strings.Contains(fmt.Sprintf("%s", output), "no such pool") {
		err = errors.New("No such pool")
	}
	return
}

func RunZpoolCommand(args []string) string {
	zpoolPath, err := exec.LookPath("zpool")
	if err != nil {
		log.Fatal("Could not find zpool in PATH")
	}
	cmd := exec.Command(zpoolPath, args...)
	out, _ := cmd.CombinedOutput()
	return fmt.Sprintf("%s", out)
}

func substringInSlice(str string, list []string) bool {
	for _, v := range list {
		if strings.Contains(str, v) {
			return true
		}
	}
	return false
}
