package vagrant

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/op/go-logging"

	"github.com/pulcy/quark/providers"
	"github.com/pulcy/quark/templates"
)

const (
	fileMode            = os.FileMode(0775)
	cloudConfigTemplate = "templates/cloud-config.tmpl"
	vagrantFileTemplate = "templates/Vagrantfile.tmpl"
	vagrantFileName     = "Vagrantfile"
	configTemplate      = "templates/config.rb.tmpl"
	configFileName      = "config.rb"
	userDataFileName    = "user-data"
)

type vagrantProvider struct {
	Logger        *logging.Logger
	folder        string
	instanceCount int
}

func NewProvider(logger *logging.Logger, folder string) providers.CloudProvider {
	return &vagrantProvider{
		Logger: logger,
		folder: folder,
	}
}

func (vp *vagrantProvider) ShowPlans() error {
	return maskAny(NotImplementedError)
}

func (vp *vagrantProvider) ShowRegions() error {
	return maskAny(NotImplementedError)
}

func (vp *vagrantProvider) ShowImages() error {
	return maskAny(NotImplementedError)
}

func (vp *vagrantProvider) ShowKeys() error {
	return maskAny(NotImplementedError)
}

// Create a machine instance
func (vp *vagrantProvider) CreateInstance(options *providers.CreateInstanceOptions, dnsProvider providers.DnsProvider) (providers.ClusterInstance, error) {
	return providers.ClusterInstance{}, maskAny(NotImplementedError)
}

// Create an entire cluster
func (vp *vagrantProvider) CreateCluster(options *providers.CreateClusterOptions, dnsProvider providers.DnsProvider) error {
	// Ensure folder exists
	if err := os.MkdirAll(vp.folder, fileMode|os.ModeDir); err != nil {
		return maskAny(err)
	}

	if _, err := os.Stat(filepath.Join(vp.folder, ".vagrant")); err == nil {
		return maskAny(fmt.Errorf("Vagrant in %s already exists", vp.folder))
	}

	vopts := struct {
		InstanceCount int
	}{
		InstanceCount: options.InstanceCount,
	}
	vp.instanceCount = options.InstanceCount

	// Vagrantfile
	content, err := templates.Render(vagrantFileTemplate, vopts)
	if err != nil {
		return maskAny(err)
	}
	if err := ioutil.WriteFile(filepath.Join(vp.folder, vagrantFileName), []byte(content), fileMode); err != nil {
		return maskAny(err)
	}

	// config.rb
	content, err = templates.Render(configTemplate, vopts)
	if err != nil {
		return maskAny(err)
	}
	if err := ioutil.WriteFile(filepath.Join(vp.folder, configFileName), []byte(content), fileMode); err != nil {
		return maskAny(err)
	}

	// user-data
	instanceOptions := options.NewCreateInstanceOptions()
	opts := instanceOptions.NewCloudConfigOptions()
	opts.PrivateIPv4 = "$private_ipv4"
	opts.IncludeSshKeys = true
	opts.PrivateClusterDevice = "eth1"

	content, err = templates.Render(cloudConfigTemplate, opts)
	if err != nil {
		return maskAny(err)
	}
	if err := ioutil.WriteFile(filepath.Join(vp.folder, userDataFileName), []byte(content), fileMode); err != nil {
		return maskAny(err)
	}

	// Start
	cmd := exec.Command("vagrant", "up")
	cmd.Dir = vp.folder
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return maskAny(err)
	}

	return nil
}

// Get names of instances of a cluster
func (vp *vagrantProvider) GetInstances(info *providers.ClusterInfo) ([]providers.ClusterInstance, error) {
	instances := []providers.ClusterInstance{}
	for i := 1; i <= vp.instanceCount; i++ {
		instances = append(instances, providers.ClusterInstance{
			Name:        fmt.Sprintf("core-%02d", i),
			PrivateIpv4: fmt.Sprintf("192.168.33.%d", 100+i),
			PublicIpv4:  fmt.Sprintf("192.168.33.%d", 100+i),
			PublicIpv6:  "",
		})
	}
	return instances, nil
}

// Remove all instances of a cluster
func (vp *vagrantProvider) DeleteCluster(info *providers.ClusterInfo, dnsProvider providers.DnsProvider) error {
	// Start
	cmd := exec.Command("vagrant", "destroy", "-f")
	cmd.Dir = vp.folder
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return maskAny(err)
	}

	os.RemoveAll(filepath.Join(vp.folder, ".vagrant"))

	return nil
}

func (vp *vagrantProvider) DeleteInstance(info *providers.ClusterInstanceInfo, dnsProvider providers.DnsProvider) error {
	return maskAny(NotImplementedError)
}

func (vp *vagrantProvider) ShowDomainRecords(domain string) error {
	return maskAny(NotImplementedError)
}
