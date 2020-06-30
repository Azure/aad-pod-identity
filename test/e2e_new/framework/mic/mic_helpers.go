// +build e2e_new

package mic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Azure/aad-pod-identity/test/e2e_new/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rl "k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	getTimeout = 1 * time.Minute
	getPolling = 5 * time.Second

	deleteTimeout = 1 * time.Minute
	deletePolling = 5 * time.Second
)

type GetLeaderInput struct {
	Getter framework.Getter
}

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
	}, getTimeout, getPolling).Should(BeTrue())

	return leaderPod
}

type DeleteLeaderInput struct {
	Getter  framework.Getter
	Deleter framework.Deleter
}

func DeleteLeader(input DeleteLeaderInput) {
	Expect(input.Getter).NotTo(BeNil(), "input.Getter is required for MIC.DeleteLeader")
	Expect(input.Deleter).NotTo(BeNil(), "input.Getter is required for MIC.DeleteLeader")

	leader := GetLeader(GetLeaderInput{
		Getter: input.Getter,
	})

	By(fmt.Sprintf("Deleting MIC Leader \"%s\"", leader.Name))

	Eventually(func() error {
		return input.Deleter.Delete(context.TODO(), leader)
	}, deleteTimeout, deletePolling).Should(Succeed())
}
