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
)

type ClusterMember struct {
	ClusterID string
	MachineID string
	PrivateIP string
	EtcdProxy bool
}

type ClusterMemberList []ClusterMember

func (cml ClusterMemberList) Render() string {
	data := ""
	for _, cm := range cml {
		proxy := ""
		if cm.EtcdProxy {
			proxy = " etcd-proxy"
		}
		data = data + fmt.Sprintf("%s=%s%s\n", cm.MachineID, cm.PrivateIP, proxy)
	}
	return data
}

func (cml ClusterMemberList) Find(instance ClusterInstance) (ClusterMember, error) {
	for _, cm := range cml {
		if cm.PrivateIP == instance.PrivateIpv4 {
			return cm, nil
		}
	}
	return ClusterMember{}, maskAny(NotFoundError)
}
