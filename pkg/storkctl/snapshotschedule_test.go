// +build unittest

package storkctl

import (
	"strconv"
	"testing"

	snapv1 "github.com/kubernetes-incubator/external-storage/snapshot/pkg/apis/crd/v1"
	storkv1 "github.com/libopenstorage/stork/pkg/apis/stork/v1alpha1"
	"github.com/portworx/sched-ops/k8s"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetSnapshotSchedulesNoSnapshotSchedule(t *testing.T) {
	cmdArgs := []string{"get", "snapshotschedules"}

	expected := "No resources found.\n"
	testCommon(t, cmdArgs, nil, expected, false)
}

func createSnapshotScheduleAndVerify(
	t *testing.T,
	name string,
	pvcName string,
	schedulePolicyName string,
	namespace string,
	preExecRule string,
	postExecRule string,
	enabled bool,
) {
	cmdArgs := []string{"create", "snapshotschedule", "-s", schedulePolicyName, "-n", namespace, "-p", pvcName, "--enabled", strconv.FormatBool(enabled), name}
	if preExecRule != "" {
		cmdArgs = append(cmdArgs, "--preExecRule", preExecRule)
	}
	if postExecRule != "" {
		cmdArgs = append(cmdArgs, "--postExecRule", postExecRule)
	}

	expected := "VolumeSnapshotSchedule " + name + " created successfully\n"
	testCommon(t, cmdArgs, nil, expected, false)

	// Make sure it was created correctly
	snapshot, err := k8s.Instance().GetSnapshotSchedule(name, namespace)
	require.NoError(t, err, "Error getting snapshot schedule")
	require.Equal(t, name, snapshot.Name, "SnapshotSchedule name mismatch")
	require.Equal(t, namespace, snapshot.Namespace, "SnapshotSchedule namespace mismatch")
	require.Equal(t, preExecRule, snapshot.Spec.PreExecRule, "SnapshotSchedule preExecRule mismatch")
	require.Equal(t, postExecRule, snapshot.Spec.PostExecRule, "SnapshotSchedule postExecRule mismatch")
}

func TestGetSnapshotSchedulesOneSnapshotSchedule(t *testing.T) {
	defer resetTest()
	createSnapshotScheduleAndVerify(t, "getsnapshotscheduletest", "pvcname", "testpolicy", "test", "preExec", "postExec", true)

	expected := "NAME                       POLICYNAME   CLUSTERPAIR    LAST-SUCCESS-TIME\n" +
		"getsnapshotscheduletest   testpolicy   clusterpair1   \n"

	cmdArgs := []string{"get", "snapshotschedules", "-n", "test"}
	testCommon(t, cmdArgs, nil, expected, false)
}

func TestGetSnapshotSchedulesMultiple(t *testing.T) {
	defer resetTest()
	createSnapshotScheduleAndVerify(t, "getsnapshotscheduletest1", "pvcname", "testpolicy", "test", "preExec", "postExec", true)
	createSnapshotScheduleAndVerify(t, "getsnapshotscheduletest2", "pvcname", "testpolicy", "test", "preExec", "postExec", true)

	expected := "NAME                        POLICYNAME   CLUSTERPAIR    LAST-SUCCESS-TIME\n" +
		"getsnapshotscheduletest1   testpolicy   clusterpair1   \n" +
		"getsnapshotscheduletest2   testpolicy   clusterpair2   \n"

	cmdArgs := []string{"get", "snapshotschedules", "getsnapshotscheduletest1", "getsnapshotscheduletest2"}
	testCommon(t, cmdArgs, nil, expected, false)

	// Should get all snapshotschedules if no name given
	cmdArgs = []string{"get", "snapshotschedules"}
	testCommon(t, cmdArgs, nil, expected, false)

	expected = "NAME                        POLICYNAME   CLUSTERPAIR    LAST-SUCCESS-TIME\n" +
		"getsnapshotscheduletest1   testpolicy   clusterpair1   \n"
	// Should get only one snapshot if name given
	cmdArgs = []string{"get", "snapshotschedules", "getsnapshotscheduletest1"}
	testCommon(t, cmdArgs, nil, expected, false)
}

func TestGetSnapshotSchedulesWithPVC(t *testing.T) {
	defer resetTest()
	createSnapshotScheduleAndVerify(t, "getsnapshotscheduletest1", "pvcname1", "testpolicy", "test", "preExec", "postExec", true)
	createSnapshotScheduleAndVerify(t, "getsnapshotscheduletest2", "pvcname2", "testpolicy", "test", "preExec", "postExec", true)

	expected := "NAME                        POLICYNAME   CLUSTERPAIR    LAST-SUCCESS-TIME\n" +
		"getsnapshotscheduletest1   testpolicy   clusterpair1   \n"

	cmdArgs := []string{"get", "snapshotschedules", "-p", "pvcname1"}
	testCommon(t, cmdArgs, nil, expected, false)
}

func TestGetSnapshotSchedulesWithStatus(t *testing.T) {
	defer resetTest()
	createSnapshotScheduleAndVerify(t, "getsnapshotschedulestatustest", "pvcname1", "testpolicy", "test", "preExec", "postExec", true)
	snapshotSchedule, err := k8s.Instance().GetSnapshotSchedule("getsnapshotschedulestatustest", "default")
	require.NoError(t, err, "Error getting snapshot")

	// Update the status of the daily snapshot
	snapshotSchedule.Status.Items = make(map[storkv1.SchedulePolicyType][]*storkv1.ScheduledVolumeSnapshotStatus)
	snapshotSchedule.Status.Items[storkv1.SchedulePolicyTypeDaily] = make([]*storkv1.ScheduledVolumeSnapshotStatus, 0)
	now := metav1.Now()
	snapshotSchedule.Status.Items[storkv1.SchedulePolicyTypeDaily] = append(snapshotSchedule.Status.Items[storkv1.SchedulePolicyTypeDaily],
		&storkv1.ScheduledVolumeSnapshotStatus{
			Name:              "dailysnapshot",
			CreationTimestamp: now,
			FinishTimestamp:   now,
			Status:            snapv1.VolumeSnapshotConditionReady,
		},
	)
	snapshotSchedule, err = k8s.Instance().UpdateSnapshotSchedule(snapshotSchedule)

	expected := "NAME                             POLICYNAME   CLUSTERPAIR    LAST-SUCCESS-TIME\n" +
		"getsnapshotschedulestatustest   testpolicy   clusterpair1   " + toTimeString(now.Time) + "\n"
	cmdArgs := []string{"get", "snapshotschedules", "getsnapshotschedulestatustest"}
	testCommon(t, cmdArgs, nil, expected, false)

	now = metav1.Now()
	snapshotSchedule.Status.Items[storkv1.SchedulePolicyTypeWeekly] = append(snapshotSchedule.Status.Items[storkv1.SchedulePolicyTypeWeekly],
		&storkv1.ScheduledVolumeSnapshotStatus{
			Name:              "weeklysnapshot",
			CreationTimestamp: now,
			FinishTimestamp:   now,
			Status:            snapv1.VolumeSnapshotConditionReady,
		},
	)
	snapshotSchedule, err = k8s.Instance().UpdateSnapshotSchedule(snapshotSchedule)

	expected = "NAME                             POLICYNAME   CLUSTERPAIR    LAST-SUCCESS-TIME\n" +
		"getsnapshotschedulestatustest   testpolicy   clusterpair1   " + toTimeString(now.Time) + "\n"
	cmdArgs = []string{"get", "snapshotschedules", "getsnapshotschedulestatustest"}
	testCommon(t, cmdArgs, nil, expected, false)

	now = metav1.Now()
	snapshotSchedule.Status.Items[storkv1.SchedulePolicyTypeMonthly] = append(snapshotSchedule.Status.Items[storkv1.SchedulePolicyTypeMonthly],
		&storkv1.ScheduledVolumeSnapshotStatus{
			Name:              "monthlysnapshot",
			CreationTimestamp: now,
			FinishTimestamp:   now,
			Status:            snapv1.VolumeSnapshotConditionReady,
		},
	)
	snapshotSchedule, err = k8s.Instance().UpdateSnapshotSchedule(snapshotSchedule)

	expected = "NAME                             POLICYNAME   CLUSTERPAIR    LAST-SUCCESS-TIME\n" +
		"getsnapshotschedulestatustest   testpolicy   clusterpair1   " + toTimeString(now.Time) + "\n"
	cmdArgs = []string{"get", "snapshotschedules", "getsnapshotschedulestatustest"}
	testCommon(t, cmdArgs, nil, expected, false)
}

func TestCreateSnapshotSchedulesNoNamespace(t *testing.T) {
	cmdArgs := []string{"create", "snapshotschedules", "-c", "clusterPair1", "snapshot1"}

	expected := "error: Need to provide atleast one namespace to migrate"
	testCommon(t, cmdArgs, nil, expected, true)
}

func TestCreateSnapshotSchedulesNoClusterPair(t *testing.T) {
	cmdArgs := []string{"create", "snapshotschedules", "snapshot1"}

	expected := "error: ClusterPair name needs to be provided for snapshot schedule"
	testCommon(t, cmdArgs, nil, expected, true)
}

func TestCreateSnapshotSchedulesNoName(t *testing.T) {
	cmdArgs := []string{"create", "snapshotschedules"}

	expected := "error: Exactly one name needs to be provided for snapshot schedule name"
	testCommon(t, cmdArgs, nil, expected, true)
}

func TestCreateSnapshotSchedules(t *testing.T) {
	defer resetTest()
	createSnapshotScheduleAndVerify(t, "createsnapshotschedule", "pvcname1", "testpolicy", "test", "preExec", "postExec", true)
}

func TestCreateDuplicateSnapshotSchedules(t *testing.T) {
	defer resetTest()
	createSnapshotScheduleAndVerify(t, "createsnapshotschedule", "pvcname1", "testpolicy", "test", "preExec", "postExec", true)
	cmdArgs := []string{"create", "snapshotschedules", "createsnapshotschedule", "-p", "pvcname1", "-s", "testpolicy", "-n", "test", "--preExecRule", "preExec", "--postExecRule", "postExec", "--enabled", "true"}

	expected := "Error from server (AlreadyExists): snapshotschedules.stork.libopenstorage.org \"createsnapshotschedule\" already exists"
	testCommon(t, cmdArgs, nil, expected, true)
}

func TestDeleteSnapshotSchedulesNoSnapshotName(t *testing.T) {
	cmdArgs := []string{"delete", "snapshotschedules"}

	expected := "error: At least one argument needs to be provided for snapshot schedule name if pvc isn't provided"
	testCommon(t, cmdArgs, nil, expected, true)
}

func TestDeleteSnapshotSchedulesNoSnapshot(t *testing.T) {
	cmdArgs := []string{"delete", "snapshotschedules", "-c", "snapshot1"}

	expected := "No resources found.\n"
	testCommon(t, cmdArgs, nil, expected, false)
}

func TestDeleteSnapshotSchedules(t *testing.T) {
	defer resetTest()
	createSnapshotScheduleAndVerify(t, "deletesnapshotschedule", "pvcname1", "testpolicy", "test", "preExec", "postExec", false)

	cmdArgs := []string{"delete", "snapshotschedules", "deletesnapshotschedule"}
	expected := "VolumeSnapshotSchedule deletesnapshotschedule deleted successfully\n"
	testCommon(t, cmdArgs, nil, expected, false)

	cmdArgs = []string{"delete", "snapshotschedules", "deletesnapshotschedule"}
	expected = "Error from server (NotFound): volumesnapshotschedules.stork.libopenstorage.org \"deletesnapshotschedule\" not found"
	testCommon(t, cmdArgs, nil, expected, true)

	createSnapshotScheduleAndVerify(t, "deletesnapshotschedule1", "pvcname1", "testpolicy", "test", "preExec", "postExec", false)
	createSnapshotScheduleAndVerify(t, "deletesnapshotschedule2", "pvcname2", "testpolicy", "test", "preExec", "postExec", false)

	cmdArgs = []string{"delete", "snapshotschedules", "deletesnapshotschedule1", "deletesnapshotschedule2"}
	expected = "VolumeSnapshotSchedule deletesnapshotschedule1 deleted successfully\n"
	expected += "VolumeSnapshotSchedule deletesnapshotschedule2 deleted successfully\n"
	testCommon(t, cmdArgs, nil, expected, false)

	createSnapshotScheduleAndVerify(t, "deletesnapshotschedule1", "pvcname1", "testpolicy", "test", "preExec", "postExec", false)
	createSnapshotScheduleAndVerify(t, "deletesnapshotschedule2", "pvcname1", "testpolicy", "test", "preExec", "postExec", false)

	cmdArgs = []string{"delete", "snapshotschedules", "-p", "pvcname1"}
	expected = "VolumeSnapshotSchedule deletesnapshotschedule1 deleted successfully\n"
	expected += "VolumeSnapshotSchedule deletesnapshotschedule2 deleted successfully\n"
	testCommon(t, cmdArgs, nil, expected, false)
}
