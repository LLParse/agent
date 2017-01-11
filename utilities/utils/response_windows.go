//+build windows

package utils

import (
	"bufio"
	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/patrickmn/go-cache"
	"github.com/rancher/agent/utilities/docker"
	"golang.org/x/net/context"
	"regexp"
	"strings"
	"time"
)

func getIP(inspect types.ContainerJSON, cache *cache.Cache) (string, error) {
	containerID := inspect.ID
	client := docker.GetClient(docker.DefaultVersion)
	execConfig := types.ExecConfig{
		AttachStdout: true,
		AttachStdin:  true,
		AttachStderr: true,
		Privileged:   true,
		Tty:          false,
		Detach:       false,
		Cmd:          []string{"powershell", "ipconfig"},
	}
	ip := ""
	// waiting for the DHCP to assign IP address. Testing purpose. May try multiple times until ip address arrives
	time.Sleep(time.Duration(2) * time.Second)
	execObj, err := client.ContainerExecCreate(context.Background(), containerID, execConfig)
	if err != nil {
		logrus.Error(err)
		return "", nil
	}
	hijack, err := client.ContainerExecAttach(context.Background(), execObj.ID, execConfig)
	if err != nil {
		logrus.Error(err)
		return "", nil
	}
	scanner := bufio.NewScanner(hijack.Reader)
	for scanner.Scan() {
		output := scanner.Text()
		if strings.Contains(output, "IPv4 Address") {
			ip = regexp.MustCompile("(?:[0-9]{1,3}\\.){3}[0-9]{1,3}$").FindString(output)
		}
	}
	hijack.Close()
	return ip, nil
}

func setupDNS(containerID string, gateway string) {
	createAndStart(containerID, []string{"powershell", "route", "ADD", "10.41.41.41", "MASK", "255.255.255.255", gateway})
	createAndStart(containerID, []string{"powershell", "Get-NetAdapter", "|", "Set-DnsClientServerAddress", "-ServerAddresses", "('10.41.41.41')"})
}

func createAndStart(containerID string, command []string) {
	client := docker.GetClient(docker.DefaultVersion)
	execConfig := types.ExecConfig{
		AttachStdout: true,
		AttachStdin:  true,
		AttachStderr: true,
		Privileged:   true,
		Tty:          false,
		Detach:       false,
		Cmd:          command,
	}

	execObj, err := client.ContainerExecCreate(context.Background(), containerID, execConfig)
	if err != nil {
		logrus.Error(err)
	}

	err = client.ContainerExecStart(context.Background(), execObj.ID, types.ExecStartCheck{})
	if err != nil {
		logrus.Error(err)
	}	
}
