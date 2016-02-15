package providers

import (
	"sync"

	"github.com/op/go-logging"
)

// UpdateClusterMembers updates /etc/cluster-members on all instances of the cluster
func UpdateClusterMembers(log *logging.Logger, info ClusterInfo, isEtcdProxy func(ClusterInstance) bool, provider CloudProvider) error {
	// See if there are already instances for the given cluster
	instances, err := provider.GetInstances(&info)
	if err != nil {
		return maskAny(err)
	}

	// Update existing members
	clusterMembers, err := loadClusterMembers(log, instances, isEtcdProxy)
	if err != nil {
		return maskAny(err)
	}

	// Now update all members in parallel
	wg := sync.WaitGroup{}
	errorChannel := make(chan error, len(instances))
	for _, i := range instances {
		wg.Add(1)
		go func(i ClusterInstance) {
			defer wg.Done()
			if err := i.UpdateClusterMembers(log, clusterMembers); err != nil {
				errorChannel <- maskAny(err)
			}
		}(i)
	}
	wg.Wait()
	close(errorChannel)
	for err := range errorChannel {
		return maskAny(err)
	}
	return nil
}

func loadClusterMembers(log *logging.Logger, instances []ClusterInstance, isEtcdProxy func(ClusterInstance) bool) ([]ClusterMember, error) {
	clusterMemberChannel := make(chan ClusterMember, len(instances))
	errorChannel := make(chan error, len(instances))
	wg := sync.WaitGroup{}
	for _, i := range instances {
		wg.Add(1)
		go func(i ClusterInstance) {
			defer wg.Done()
			machineID, err := i.GetMachineID(log)
			if err != nil {
				errorChannel <- maskAny(err)
				return
			}
			etcdProxy, err := i.IsEtcdProxy(log)
			if err != nil {
				errorChannel <- maskAny(err)
				return
			}
			if isEtcdProxy(i) {
				etcdProxy = true
			}
			clusterMemberChannel <- ClusterMember{
				MachineID: machineID,
				PrivateIP: i.PrivateIpv4,
				EtcdProxy: etcdProxy,
			}
		}(i)
	}
	wg.Wait()
	close(clusterMemberChannel)
	close(errorChannel)

	for err := range errorChannel {
		return nil, maskAny(err)
	}

	result := []ClusterMember{}
	for cm := range clusterMemberChannel {
		result = append(result, cm)
	}
	return result, nil
}