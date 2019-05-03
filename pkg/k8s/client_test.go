package k8s

import (
	"testing"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetSecret(t *testing.T) {
	secretName := "clientSecret"

	fakeClient := fake.NewSimpleClientset()

	secret := &v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName}}
	fakeClient.CoreV1().Secrets("default").Create(secret)

	kubeClient := &KubeClient{ClientSet: fakeClient}

	secretRef := &v1.SecretReference{
		Name:      secretName,
		Namespace: "default",
	}
	retrievedSecret, err := kubeClient.GetSecret(secretRef)
	if err != nil {
		t.Fatalf("Error getting secret: %v", err)
	}
	if retrievedSecret.ObjectMeta.Name != secretName {
		t.Fatalf("Incorrect secret name: %v", retrievedSecret.ObjectMeta.Name)
	}
}

func TestGetPodName(t *testing.T) {
	podIP := "10.0.0.8"

	fakeClient := fake.NewSimpleClientset()

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testpodname",
			Namespace: "default",
		},
		Status: v1.PodStatus{
			PodIP: podIP,
		},
	}
	fakeClient.CoreV1().Pods("default").Create(pod)

	kubeClient := &KubeClient{ClientSet: fakeClient}

	podNs, podName, err := kubeClient.GetPodName(podIP)
	if err != nil {
		t.Fatalf("Error getting pod: %v", err)
	}
	if podName != "testpodname" {
		t.Fatalf("Incorrect pod name: %v", podName)
	}
	if podNs != "default" {
		t.Fatalf("Incorrect pod ns: %v", podNs)
	}
}

func TestPodListRetries(t *testing.T) {
	// this test is to solely test the retry and sleep logic works as expected
	podIP := "10.0.0.8"

	fakeClient := fake.NewSimpleClientset()

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testpodname1",
			Namespace: "default",
		},
		Status: v1.PodStatus{
			PodIP: podIP,
		},
	}
	kubeClient := &KubeClient{ClientSet: fakeClient}

	time.AfterFunc(time.Duration(1200*time.Millisecond), func() {
		fakeClient.CoreV1().Pods("default").Create(pod)
	})

	start := time.Now()
	podNs, podName, err := kubeClient.GetPodName(podIP)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Error getting pod: %v", err)
	}
	if podName != "testpodname1" {
		t.Fatalf("Incorrect pod name: %v", podName)
	}
	if podNs != "default" {
		t.Fatalf("Incorrect pod ns: %v", podNs)
	}
	// check the retries actually work as the pod object is created only after 1.2s
	if elapsed < 1200*time.Millisecond {
		t.Fatalf("Retry logic not working as expected. Elapsed time: %v", elapsed)
	}
}
