/*
Copyright 2017 the Heptio Ark contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	arkv1api "github.com/heptio/ark/pkg/apis/ark/v1"
	"github.com/heptio/ark/pkg/cmd/util/downloadrequest"
	clientset "github.com/heptio/ark/pkg/generated/clientset/versioned"
	"github.com/heptio/ark/pkg/volume"
)

// DescribeBackup describes a backup in human-readable format.
func DescribeBackup(
	backup *arkv1api.Backup,
	deleteRequests []arkv1api.DeleteBackupRequest,
	podVolumeBackups []arkv1api.PodVolumeBackup,
	details bool,
	arkClient clientset.Interface,
) string {
	return Describe(func(d *Describer) {
		d.DescribeMetadata(backup.ObjectMeta)

		d.Println()
		phase := backup.Status.Phase
		if phase == "" {
			phase = arkv1api.BackupPhaseNew
		}
		d.Printf("Phase:\t%s\n", phase)

		d.Println()
		DescribeBackupSpec(d, backup.Spec)

		d.Println()
		DescribeBackupStatus(d, backup, details, arkClient)

		if len(deleteRequests) > 0 {
			d.Println()
			DescribeDeleteBackupRequests(d, deleteRequests)
		}

		if len(podVolumeBackups) > 0 {
			d.Println()
			DescribePodVolumeBackups(d, podVolumeBackups, details)
		}
	})
}

// DescribeBackupSpec describes a backup spec in human-readable format.
func DescribeBackupSpec(d *Describer, spec arkv1api.BackupSpec) {
	// TODO make a helper for this and use it in all the describers.
	d.Printf("Namespaces:\n")
	var s string
	if len(spec.IncludedNamespaces) == 0 {
		s = "*"
	} else {
		s = strings.Join(spec.IncludedNamespaces, ", ")
	}
	d.Printf("\tIncluded:\t%s\n", s)
	if len(spec.ExcludedNamespaces) == 0 {
		s = "<none>"
	} else {
		s = strings.Join(spec.ExcludedNamespaces, ", ")
	}
	d.Printf("\tExcluded:\t%s\n", s)

	d.Println()
	d.Printf("Resources:\n")
	if len(spec.IncludedResources) == 0 {
		s = "*"
	} else {
		s = strings.Join(spec.IncludedResources, ", ")
	}
	d.Printf("\tIncluded:\t%s\n", s)
	if len(spec.ExcludedResources) == 0 {
		s = "<none>"
	} else {
		s = strings.Join(spec.ExcludedResources, ", ")
	}
	d.Printf("\tExcluded:\t%s\n", s)

	d.Printf("\tCluster-scoped:\t%s\n", BoolPointerString(spec.IncludeClusterResources, "excluded", "included", "auto"))

	d.Println()
	s = "<none>"
	if spec.LabelSelector != nil {
		s = metav1.FormatLabelSelector(spec.LabelSelector)
	}
	d.Printf("Label selector:\t%s\n", s)

	d.Println()
	d.Printf("Storage Location:\t%s\n", spec.StorageLocation)

	d.Println()
	d.Printf("Snapshot PVs:\t%s\n", BoolPointerString(spec.SnapshotVolumes, "false", "true", "auto"))

	d.Println()
	d.Printf("TTL:\t%s\n", spec.TTL.Duration)

	d.Println()
	if len(spec.Hooks.Resources) == 0 {
		d.Printf("Hooks:\t<none>\n")
	} else {
		d.Printf("Hooks:\n")
		d.Printf("\tResources:\n")
		for _, backupResourceHookSpec := range spec.Hooks.Resources {
			d.Printf("\t\t%s:\n", backupResourceHookSpec.Name)
			d.Printf("\t\t\tNamespaces:\n")
			var s string
			if len(spec.IncludedNamespaces) == 0 {
				s = "*"
			} else {
				s = strings.Join(spec.IncludedNamespaces, ", ")
			}
			d.Printf("\t\t\t\tIncluded:\t%s\n", s)
			if len(spec.ExcludedNamespaces) == 0 {
				s = "<none>"
			} else {
				s = strings.Join(spec.ExcludedNamespaces, ", ")
			}
			d.Printf("\t\t\t\tExcluded:\t%s\n", s)

			d.Println()
			d.Printf("\t\t\tResources:\n")
			if len(spec.IncludedResources) == 0 {
				s = "*"
			} else {
				s = strings.Join(spec.IncludedResources, ", ")
			}
			d.Printf("\t\t\t\tIncluded:\t%s\n", s)
			if len(spec.ExcludedResources) == 0 {
				s = "<none>"
			} else {
				s = strings.Join(spec.ExcludedResources, ", ")
			}
			d.Printf("\t\t\t\tExcluded:\t%s\n", s)

			d.Println()
			s = "<none>"
			if backupResourceHookSpec.LabelSelector != nil {
				s = metav1.FormatLabelSelector(backupResourceHookSpec.LabelSelector)
			}
			d.Printf("\t\t\tLabel selector:\t%s\n", s)

			for _, hook := range backupResourceHookSpec.Hooks {
				if hook.Exec != nil {
					d.Println()
					d.Printf("\t\t\tExec Hook:\n")
					d.Printf("\t\t\t\tContainer:\t%s\n", hook.Exec.Container)
					d.Printf("\t\t\t\tCommand:\t%s\n", strings.Join(hook.Exec.Command, " "))
					d.Printf("\t\t\t\tOn Error:\t%s\n", hook.Exec.OnError)
					d.Printf("\t\t\t\tTimeout:\t%s\n", hook.Exec.Timeout.Duration)
				}
			}
		}
	}

}

// DescribeBackupStatus describes a backup status in human-readable format.
func DescribeBackupStatus(d *Describer, backup *arkv1api.Backup, details bool, arkClient clientset.Interface) {
	status := backup.Status

	d.Printf("Backup Format Version:\t%d\n", status.Version)

	d.Println()
	// "<n/a>" output should only be applicable for backups that failed validation
	if status.StartTimestamp.Time.IsZero() {
		d.Printf("Started:\t%s\n", "<n/a>")
	} else {
		d.Printf("Started:\t%s\n", status.StartTimestamp.Time)
	}
	if status.CompletionTimestamp.Time.IsZero() {
		d.Printf("Completed:\t%s\n", "<n/a>")
	} else {
		d.Printf("Completed:\t%s\n", status.CompletionTimestamp.Time)
	}

	d.Println()
	d.Printf("Expiration:\t%s\n", status.Expiration.Time)
	d.Println()

	d.Printf("Validation errors:")
	if len(status.ValidationErrors) == 0 {
		d.Printf("\t<none>\n")
	} else {
		for _, ve := range status.ValidationErrors {
			d.Printf("\t%s\n", ve)
		}
	}

	d.Println()
	if len(status.VolumeBackups) > 0 {
		// pre-v0.10 backup
		d.Printf("Persistent Volumes:\n")
		for pvName, info := range status.VolumeBackups {
			printSnapshot(d, pvName, info.SnapshotID, info.Type, info.AvailabilityZone, info.Iops)
		}
		return
	}

	if status.VolumeSnapshotsAttempted > 0 {
		// v0.10+ backup
		if !details {
			d.Printf("Persistent Volumes:\t%d of %d snapshots completed successfully (specify --details for more information)\n", status.VolumeSnapshotsCompleted, status.VolumeSnapshotsAttempted)
			return
		}

		buf := new(bytes.Buffer)
		if err := downloadrequest.Stream(arkClient.ArkV1(), backup.Namespace, backup.Name, arkv1api.DownloadTargetKindBackupVolumeSnapshots, buf, downloadRequestTimeout); err != nil {
			d.Printf("Persistent Volumes:\t<error getting volume snapshot info: %v>\n", err)
			return
		}

		var snapshots []*volume.Snapshot
		if err := json.NewDecoder(buf).Decode(&snapshots); err != nil {
			d.Printf("Persistent Volumes:\t<error reading volume snapshot info: %v>\n", err)
			return
		}

		d.Printf("Persistent Volumes:\n")
		for _, snap := range snapshots {
			printSnapshot(d, snap.Spec.PersistentVolumeName, snap.Status.ProviderSnapshotID, snap.Spec.VolumeType, snap.Spec.VolumeAZ, snap.Spec.VolumeIOPS)
		}
		return
	}

	d.Printf("Persistent Volumes: <none included>\n")
}

func printSnapshot(d *Describer, pvName, snapshotID, volumeType, volumeAZ string, iops *int64) {
	d.Printf("\t%s:\n", pvName)
	d.Printf("\t\tSnapshot ID:\t%s\n", snapshotID)
	d.Printf("\t\tType:\t%s\n", volumeType)
	d.Printf("\t\tAvailability Zone:\t%s\n", volumeAZ)
	iopsString := "<N/A>"
	if iops != nil {
		iopsString = fmt.Sprintf("%d", *iops)
	}
	d.Printf("\t\tIOPS:\t%s\n", iopsString)
}

// DescribeDeleteBackupRequests describes delete backup requests in human-readable format.
func DescribeDeleteBackupRequests(d *Describer, requests []arkv1api.DeleteBackupRequest) {
	d.Printf("Deletion Attempts")
	if count := failedDeletionCount(requests); count > 0 {
		d.Printf(" (%d failed)", count)
	}
	d.Println(":")

	started := false
	for _, req := range requests {
		if !started {
			started = true
		} else {
			d.Println()
		}

		d.Printf("\t%s: %s\n", req.CreationTimestamp.String(), req.Status.Phase)
		if len(req.Status.Errors) > 0 {
			d.Printf("\tErrors:\n")
			for _, err := range req.Status.Errors {
				d.Printf("\t\t%s\n", err)
			}
		}
	}
}

func failedDeletionCount(requests []arkv1api.DeleteBackupRequest) int {
	var count int
	for _, req := range requests {
		if req.Status.Phase == arkv1api.DeleteBackupRequestPhaseProcessed && len(req.Status.Errors) > 0 {
			count++
		}
	}
	return count
}

// DescribePodVolumeBackups describes pod volume backups in human-readable format.
func DescribePodVolumeBackups(d *Describer, backups []arkv1api.PodVolumeBackup, details bool) {
	if details {
		d.Printf("Restic Backups:\n")
	} else {
		d.Printf("Restic Backups (specify --details for more information):\n")
	}

	// separate backups by phase (combining <none> and New into a single group)
	backupsByPhase := groupByPhase(backups)

	// go through phases in a specific order
	for _, phase := range []string{
		string(arkv1api.PodVolumeBackupPhaseCompleted),
		string(arkv1api.PodVolumeBackupPhaseFailed),
		"In Progress",
		string(arkv1api.PodVolumeBackupPhaseNew),
	} {
		if len(backupsByPhase[phase]) == 0 {
			continue
		}

		// if we're not printing details, just report the phase and count
		if !details {
			d.Printf("\t%s:\t%d\n", phase, len(backupsByPhase[phase]))
			continue
		}

		// group the backups in the current phase by pod (i.e. "ns/name")
		backupsByPod := new(volumesByPod)

		for _, backup := range backupsByPhase[phase] {
			backupsByPod.Add(backup.Spec.Pod.Namespace, backup.Spec.Pod.Name, backup.Spec.Volume)
		}

		d.Printf("\t%s:\n", phase)
		for _, backupGroup := range backupsByPod.Sorted() {
			sort.Strings(backupGroup.volumes)

			// print volumes backed up for this pod
			d.Printf("\t\t%s: %s\n", backupGroup.label, strings.Join(backupGroup.volumes, ", "))
		}
	}
}

func groupByPhase(backups []arkv1api.PodVolumeBackup) map[string][]arkv1api.PodVolumeBackup {
	backupsByPhase := make(map[string][]arkv1api.PodVolumeBackup)

	phaseToGroup := map[arkv1api.PodVolumeBackupPhase]string{
		arkv1api.PodVolumeBackupPhaseCompleted:  string(arkv1api.PodVolumeBackupPhaseCompleted),
		arkv1api.PodVolumeBackupPhaseFailed:     string(arkv1api.PodVolumeBackupPhaseFailed),
		arkv1api.PodVolumeBackupPhaseInProgress: "In Progress",
		arkv1api.PodVolumeBackupPhaseNew:        string(arkv1api.PodVolumeBackupPhaseNew),
		"":                                      string(arkv1api.PodVolumeBackupPhaseNew),
	}

	for _, backup := range backups {
		group := phaseToGroup[backup.Status.Phase]
		backupsByPhase[group] = append(backupsByPhase[group], backup)
	}

	return backupsByPhase
}

type podVolumeGroup struct {
	label   string
	volumes []string
}

// volumesByPod stores podVolumeGroups, where the grouping
// label is "namespace/name".
type volumesByPod struct {
	volumesByPodMap   map[string]*podVolumeGroup
	volumesByPodSlice []*podVolumeGroup
}

// Add adds a pod volume with the specified pod namespace, name
// and volume to the appropriate group.
func (v *volumesByPod) Add(namespace, name, volume string) {
	if v.volumesByPodMap == nil {
		v.volumesByPodMap = make(map[string]*podVolumeGroup)
	}

	key := fmt.Sprintf("%s/%s", namespace, name)

	if group, ok := v.volumesByPodMap[key]; !ok {
		group := &podVolumeGroup{
			label:   key,
			volumes: []string{volume},
		}

		v.volumesByPodMap[key] = group
		v.volumesByPodSlice = append(v.volumesByPodSlice, group)
	} else {
		group.volumes = append(group.volumes, volume)
	}
}

// Sorted returns a slice of all pod volume groups, ordered by
// label.
func (v *volumesByPod) Sorted() []*podVolumeGroup {
	sort.Slice(v.volumesByPodSlice, func(i, j int) bool {
		return v.volumesByPodSlice[i].label <= v.volumesByPodSlice[j].label
	})

	return v.volumesByPodSlice
}
