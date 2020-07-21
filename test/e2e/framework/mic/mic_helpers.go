// +build e2e

package mic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/aad-pod-identity/test/e2e/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rl "k8s.io/client-go/tools/leaderelection/resourcelock"
)

// GetLeaderInput is the input for GetLeader.
type GetLeaderInput struct {
	Getter framework.Getter
}

// GetLeader returns the MIC pod which won the leader election.
func GetLeader(input GetLeaderInput) *corev1.Pod {
	Expect(input.Getter).NotTo(BeNil(), "input.Getter is required for MIC.GetLeader")

	By("Getting MIC Leader")

	leaderPod := &corev1.Pod{}
	Eventually(func() (bool, error) {
		endpoints := &corev1.Endpoints{}
		if err := input.Getter.Get(context.TODO(), client.ObjectKey{Name: "aad-pod-identity-mic", Namespace: corev1.NamespaceDefault}, endpoints); err != nil {
			return false, err
		}

		leRecord := &rl.LeaderElectionRecord{}
		err := json.Unmarshal([]byte(endpoints.ObjectMeta.Annotations["control-plane.alpha.kubernetes.io/leader"]), leRecord)
		if err != nil {
			return false, err
		}

		leaderName := leRecord.HolderIdentity
		if err := input.Getter.Get(context.TODO(), client.ObjectKey{Name: leaderName, Namespace: corev1.NamespaceDefault}, leaderPod); err != nil {
			return false, err
		}

		return true, nil
	}, framework.GetTimeout, framework.GetPolling).Should(BeTrue())

	return leaderPod
}

// DeleteLeaderInput is the input for DeleteLeader.
type DeleteLeaderInput struct {
	Getter  framework.Getter
	Deleter framework.Deleter
}

// DeleteLeader deletes the MIC pod which won the leader election.
func DeleteLeader(input DeleteLeaderInput) {
	Expect(input.Getter).NotTo(BeNil(), "input.Getter is required for MIC.DeleteLeader")
	Expect(input.Deleter).NotTo(BeNil(), "input.Deleter is required for MIC.DeleteLeader")

	leader := GetLeader(GetLeaderInput{
		Getter: input.Getter,
	})

	By(fmt.Sprintf("Deleting MIC Leader \"%s\"", leader.Name))

	Eventually(func() error {
		return input.Deleter.Delete(context.TODO(), leader)
	}, framework.DeleteTimeout, framework.DeletePolling).Should(Succeed())
}
