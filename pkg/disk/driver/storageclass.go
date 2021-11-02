/*
Copyright (C) 2018 Yunify, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this work except in compliance with the License.
You may obtain a copy of the License in the LICENSE file, or at:

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package driver

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/yunify/qingcloud-csi/pkg/common"
	"k8s.io/klog"
)

const (
	StorageClassTypeName    = "type"
	StorageClassFsTypeName  = "fsType"
	StorageClassReplicaName = "replica"
	StorageClassTagsName    = "tags"
)

type QingStorageClass struct {
	diskType VolumeType
	fsType   string
	replica  int
	tags     []string
}

// NewDefaultQingStorageClassFromType create default qingStorageClass by specified volume type
func NewDefaultQingStorageClassFromType(diskType VolumeType) *QingStorageClass {
	if diskType.IsValid() != true {
		return nil
	}
	return &QingStorageClass{
		diskType: diskType,
		fsType:   common.DefaultFileSystem,
		replica:  DefaultDiskReplicaType,
	}
}

// NewQingStorageClassFromMap create qingStorageClass object from map
func NewQingStorageClassFromMap(opt map[string]string, topology *Topology) (*QingStorageClass, error) {
	volType := -1
	fsType := ""
	replica := -1
	var tags []string
	for k, v := range opt {
		switch strings.ToLower(k) {
		case strings.ToLower(StorageClassTypeName):
			// Convert to integer
			iv, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			volType = iv
		case strings.ToLower(StorageClassFsTypeName):
			if len(v) != 0 && !IsValidFileSystemType(v) {
				return nil, fmt.Errorf("unsupported filesystem type %s", v)
			}
			fsType = v
		case strings.ToLower(StorageClassReplicaName):
			iv, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			replica = iv
		case strings.ToLower(StorageClassTagsName):
			if len(v) > 0 {
				tags = strings.Split(strings.ReplaceAll(v, " ", ""), ",")
			}
		}
	}

	var t VolumeType
	if volType == -1 {
		t = DefaultVolumeType
		if topology != nil {
			preferredVolumeType, ok := InstanceTypeAttachPreferred[topology.GetInstanceType()]
			if ok {
				t = preferredVolumeType
			} else {
				klog.Infof("failed to get instance type %d preferred volume type, fallback to use %s",
					topology.GetInstanceType(), DefaultVolumeType)
			}
		}
	} else {
		t = VolumeType(volType)
	}

	if !t.IsValid() {
		return nil, fmt.Errorf("unsupported volume type %d", volType)
	}
	sc := NewDefaultQingStorageClassFromType(t)
	_ = sc.setFsType(fsType)
	_ = sc.setReplica(replica)
	sc.setTags(tags)
	return sc, nil
}

func (sc QingStorageClass) GetDiskType() VolumeType {
	return sc.diskType
}

func (sc QingStorageClass) GetFsType() string {
	return sc.fsType
}

func (sc QingStorageClass) GetReplica() int {
	return sc.replica
}

func (sc QingStorageClass) GetTags() []string {
	return sc.tags
}

func (sc *QingStorageClass) setFsType(fs string) error {
	if !IsValidFileSystemType(fs) {
		return fmt.Errorf("unsupported filesystem type %s", fs)
	}
	sc.fsType = fs
	return nil
}

func (sc *QingStorageClass) setReplica(repl int) error {
	if !IsValidReplica(repl) {
		return fmt.Errorf("unsupported replica %d", repl)
	}
	sc.replica = repl
	return nil
}

func (sc *QingStorageClass) setTags(tagsStr []string) {
	sc.tags = tagsStr
}

// Required Volume Size
func (sc QingStorageClass) GetRequiredVolumeSizeByte(capRange *csi.CapacityRange) (int64, error) {
	if capRange == nil {
		//TODO To be verified
		return 0 * common.Gib, nil
	}
	res := int64(0)
	if capRange.GetRequiredBytes() > 0 {
		res = capRange.GetRequiredBytes()
	}
	if capRange.GetLimitBytes() > 0 && res > capRange.GetLimitBytes() {
		return -1, fmt.Errorf("volume required bytes %d greater than limit bytes %d", res, capRange.GetLimitBytes())
	}
	return res, nil
}
