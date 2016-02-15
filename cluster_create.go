package main

import (
	"github.com/spf13/cobra"

	"github.com/pulcy/quark/providers"
)

var (
	cmdCreateCluster = &cobra.Command{
		Use: "create",
		Run: createCluster,
	}

	createClusterFlags providers.CreateClusterOptions
)

func init() {
	cmdCreateCluster.Flags().StringVar(&createClusterFlags.Domain, "domain", defaultDomain(), "Cluster domain")
	cmdCreateCluster.Flags().StringVar(&createClusterFlags.Name, "name", "", "Cluster name")
	cmdCreateCluster.Flags().StringVar(&createClusterFlags.Image, "image", defaultClusterImage, "OS image to run on new instances")
	cmdCreateCluster.Flags().StringVar(&createClusterFlags.Region, "region", defaultClusterRegion(), "Region to create the instances in")
	cmdCreateCluster.Flags().StringVar(&createClusterFlags.Size, "size", defaultClusterSize, "Size of the new instances")
	cmdCreateCluster.Flags().IntVar(&createClusterFlags.InstanceCount, "instance-count", defaultInstanceCount, "Number of instances in cluster")
	cmdCreateCluster.Flags().StringVar(&createClusterFlags.GluonImage, "gluon-image", defaultGluonImage, "Image containing gluon")
	cmdCreateCluster.Flags().StringVar(&createClusterFlags.RebootStrategy, "reboot-strategy", defaultRebootStrategy, "CoreOS reboot strategy")
	cmdCreateCluster.Flags().StringVar(&createClusterFlags.PrivateRegistryUrl, "private-registry-url", defaultPrivateRegistryUrl(), "URL of private docker registry")
	cmdCreateCluster.Flags().StringVar(&createClusterFlags.PrivateRegistryUserName, "private-registry-username", defaultPrivateRegistryUserName(), "Username for private registry")
	cmdCreateCluster.Flags().StringVar(&createClusterFlags.PrivateRegistryPassword, "private-registry-password", defaultPrivateRegistryPassword(), "Password for private registry")
	cmdCreateCluster.Flags().StringSliceVar(&createClusterFlags.SSHKeyNames, "ssh-key", defaultSshKeys(), "Names of SSH keys to add to instances")
	cmdCluster.AddCommand(cmdCreateCluster)
}

func createCluster(cmd *cobra.Command, args []string) {
	clusterInfoFromArgs(&createClusterFlags.ClusterInfo, args)

	provider := newProvider()

	// Validate
	if err := createClusterFlags.Validate(); err != nil {
		Exitf("Create failed: %s\n", err.Error())
	}

	// See if there are already instances for the given cluster
	instances, err := provider.GetInstances(&createClusterFlags.ClusterInfo)
	if err != nil {
		Exitf("Failed to query existing instances: %v\n", err)
	}
	if len(instances) > 0 {
		Exitf("Cluster %s.%s already exists.\n", createClusterFlags.Name, createClusterFlags.Domain)
	}

	// Create
	err = provider.CreateCluster(&createClusterFlags, newDnsProvider())
	if err != nil {
		Exitf("Failed to create new cluster: %v\n", err)
	}

	// Update all members
	isEtcdProxy := func(i providers.ClusterInstance) bool {
		return false
	}
	if err := providers.UpdateClusterMembers(log, createClusterFlags.ClusterInfo, isEtcdProxy, provider); err != nil {
		Exitf("Failed to update cluster members: %v\n", err)
	}

	Infof("Cluster created\n")
}