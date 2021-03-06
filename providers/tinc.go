// Copyright (c) 2016 Pulcy.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package providers

import (
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/op/go-logging"
)

// ReconfigureTincCluster creates the tinc configuration on all instances of the given cluster.
func ReconfigureTincCluster(log *logging.Logger, info ClusterInfo, provider CloudProvider) error {
	// Load all instances
	instances, err := provider.GetInstances(info)
	if err != nil {
		return maskAny(err)
	}

	// Call reconfigure-tinc-host on all instances
	if instances.ReconfigureTincCluster(log); err != nil {
		return maskAny(err)
	}

	return nil
}

// ReconfigureTincCluster creates the tinc configuration on all given instances.
func (instances ClusterInstanceList) ReconfigureTincCluster(log *logging.Logger) error {
	// Now update all members in parallel
	vpnName := "pulcy"
	wg := sync.WaitGroup{}
	errorChannel := make(chan error, len(instances))
	for _, i := range instances {
		wg.Add(1)
		go func(i ClusterInstance) {
			defer wg.Done()
			if err := configureTincHost(log, i, vpnName, instances); err != nil {
				errorChannel <- maskAny(err)
			}
		}(i)
	}
	wg.Wait()
	close(errorChannel)
	for err := range errorChannel {
		return maskAny(err)
	}

	for _, i := range instances {
		if err := distributeTincHosts(log, i, vpnName, instances); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

func configureTincHost(log *logging.Logger, i ClusterInstance, vpnName string, instances ClusterInstanceList) error {
	connectTo := []string{}
	for _, x := range instances {
		if x.Name != i.Name {
			connectTo = append(connectTo, tincName(x))
		}
	}
	if err := createTincConf(log, i, vpnName, connectTo); err != nil {
		return maskAny(err)
	}
	if err := createTincHostsConf(log, i, vpnName); err != nil {
		return maskAny(err)
	}
	if err := createTincScripts(log, i, vpnName); err != nil {
		return maskAny(err)
	}
	if err := createTincService(log, i, vpnName); err != nil {
		return maskAny(err)
	}
	//Create key
	if _, err := i.runRemoteCommand(log, fmt.Sprintf("sudo tincd -n %s -K", vpnName), "", false); err != nil {
		return maskAny(err)
	}
	return nil
}

// tincName creates the name of the instance in Tinc
func tincName(i ClusterInstance) string {
	return strings.Replace(strings.Replace(i.Name, ".", "_", -1), "-", "_", -1)
}

func distributeTincHosts(log *logging.Logger, i ClusterInstance, vpnName string, instances ClusterInstanceList) error {
	conf, err := getTincHostsConf(log, i, vpnName)
	if err != nil {
		return maskAny(err)
	}
	tincName := tincName(i)
	for _, x := range instances {
		if x.Name != i.Name {
			err := setTincHostsConf(log, x, vpnName, tincName, conf)
			if err != nil {
				return maskAny(err)
			}
		}
	}
	return nil
}

// createTincConf creates a tinc.conf for the host of the given instance
func createTincConf(log *logging.Logger, i ClusterInstance, vpnName string, connectTo []string) error {
	lines := []string{
		fmt.Sprintf("Name = %s", tincName(i)),
		"AddressFamily = ipv4",
		"Interface = tun0",
	}
	for _, name := range connectTo {
		lines = append(lines, fmt.Sprintf("ConnectTo = %s", name))
	}
	confDir := path.Join("/etc/tinc", vpnName)
	confPath := path.Join(confDir, "tinc.conf")
	if _, err := i.runRemoteCommand(log, fmt.Sprintf("sudo mkdir -p %s", confDir), "", false); err != nil {
		return maskAny(err)
	}
	if _, err := i.runRemoteCommand(log, fmt.Sprintf("sudo tee %s", confPath), strings.Join(lines, "\n"), false); err != nil {
		return maskAny(err)
	}
	return nil
}

// createTincHostsConf creates a /etc/tinc/<vpnName>/hosts/<hostName> for the host of the given instance
func createTincHostsConf(log *logging.Logger, i ClusterInstance, vpnName string) error {
	lines := []string{
		fmt.Sprintf("Address = %s", i.PrivateIP),
		fmt.Sprintf("Subnet = %s/32", i.ClusterIP),
	}
	confDir := path.Join("/etc/tinc", vpnName, "hosts")
	confPath := path.Join(confDir, tincName(i))
	if _, err := i.runRemoteCommand(log, fmt.Sprintf("sudo mkdir -p %s", confDir), "", false); err != nil {
		return maskAny(err)
	}
	if _, err := i.runRemoteCommand(log, fmt.Sprintf("sudo tee %s", confPath), strings.Join(lines, "\n"), false); err != nil {
		return maskAny(err)
	}
	return nil
}

// createTincScripts creates a /etc/tinc/<vpnName>/tinc-up|down for the host of the given instance
func createTincScripts(log *logging.Logger, i ClusterInstance, vpnName string) error {
	upLines := []string{
		"#!/bin/sh",
		fmt.Sprintf("ifconfig $INTERFACE %s netmask 255.255.255.0", i.ClusterIP),
	}
	downLines := []string{
		"#!/bin/sh",
		"ifconfig $INTERFACE down",
	}
	confDir := path.Join("/etc/tinc", vpnName)
	upPath := path.Join(confDir, "tinc-up")
	downPath := path.Join(confDir, "tinc-down")
	if _, err := i.runRemoteCommand(log, fmt.Sprintf("sudo mkdir -p %s", confDir), "", false); err != nil {
		return maskAny(err)
	}
	if _, err := i.runRemoteCommand(log, fmt.Sprintf("sudo tee %s", upPath), strings.Join(upLines, "\n"), false); err != nil {
		return maskAny(err)
	}
	if _, err := i.runRemoteCommand(log, fmt.Sprintf("sudo tee %s", downPath), strings.Join(downLines, "\n"), false); err != nil {
		return maskAny(err)
	}
	if _, err := i.runRemoteCommand(log, fmt.Sprintf("sudo chmod 755 %s %s", upPath, downPath), "", false); err != nil {
		return maskAny(err)
	}
	return nil
}

// getTincHostsConf reads a /etc/tinc/<vpnName>/hosts/<hostName> for the host of the given instance
func getTincHostsConf(log *logging.Logger, i ClusterInstance, vpnName string) (string, error) {
	confDir := path.Join("/etc/tinc", vpnName, "hosts")
	confPath := path.Join(confDir, tincName(i))
	content, err := i.runRemoteCommand(log, "cat "+confPath, "", false)
	if err != nil {
		return "", maskAny(err)
	}
	return content, nil
}

// setTincHostsConf creates a /etc/tinc/<vpnName>/hosts/<hostName> from the given content
func setTincHostsConf(log *logging.Logger, i ClusterInstance, vpnName, tincName, content string) error {
	confDir := path.Join("/etc/tinc", vpnName, "hosts")
	confPath := path.Join(confDir, tincName)
	if _, err := i.runRemoteCommand(log, fmt.Sprintf("sudo mkdir -p %s", confDir), "", false); err != nil {
		return maskAny(err)
	}
	if _, err := i.runRemoteCommand(log, fmt.Sprintf("sudo tee %s", confPath), content, false); err != nil {
		return maskAny(err)
	}
	return nil
}

// createTincService creates /etc/systemd/system/tinc.service on the given instance
func createTincService(log *logging.Logger, i ClusterInstance, vpnName string) error {
	lines := []string{
		"[Unit]",
		fmt.Sprintf("Description=tinc for network %s", vpnName),
		"After=local-fs.target network-pre.target networking.service",
		"Before=network.target",
		"",
		"[Service]",
		"Type=simple",
		fmt.Sprintf("ExecStart=/usr/sbin/tincd -D -n %s", vpnName),
		fmt.Sprintf("ExecReload=/usr/sbin/tincd -n %s reload", vpnName),
		fmt.Sprintf("ExecStop=/usr/sbin/tincd -n %s stop", vpnName),
		"TimeoutStopSec=5",
		"Restart=always",
		"RestartSec=60",
		"",
		"[Install]",
		"WantedBy=multi-user.target",
	}
	confPath := "/etc/systemd/system/tinc.service"
	if _, err := i.runRemoteCommand(log, fmt.Sprintf("sudo tee %s", confPath), strings.Join(lines, "\n"), false); err != nil {
		return maskAny(err)
	}
	if _, err := i.runRemoteCommand(log, "sudo systemctl enable tinc.service", "", false); err != nil {
		return maskAny(err)
	}
	return nil
}
