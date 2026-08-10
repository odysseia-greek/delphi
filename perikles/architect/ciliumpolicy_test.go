package architect

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestResolveAppSelectorNameForCronJob(t *testing.T) {
	const namespace = "tests"

	t.Run("uses the Job pod template app label", func(t *testing.T) {
		handler := &PeriklesHandler{Kube: newFakeKubeClient()}
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{Name: "testsuite-124124", Namespace: namespace},
			Spec: batchv1.JobSpec{Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "testsuite"}},
			}},
		}
		_, err := handler.Kube.BatchV1().Jobs(namespace).Create(context.Background(), job, metav1.CreateOptions{})
		require.NoError(t, err)

		app, err := handler.resolveAppSelectorName(job.Name, namespace, "job")
		require.NoError(t, err)
		require.Equal(t, "testsuite", app)
	})

	t.Run("falls back to the owning CronJob name", func(t *testing.T) {
		handler := &PeriklesHandler{Kube: newFakeKubeClient()}
		cronJob := &batchv1.CronJob{ObjectMeta: metav1.ObjectMeta{Name: "testsuite", Namespace: namespace}}
		_, err := handler.Kube.BatchV1().CronJobs(namespace).Create(context.Background(), cronJob, metav1.CreateOptions{})
		require.NoError(t, err)

		job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{
			Name:      "testsuite-124124",
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{{
				Kind: "CronJob", Name: cronJob.Name,
			}},
		}}
		_, err = handler.Kube.BatchV1().Jobs(namespace).Create(context.Background(), job, metav1.CreateOptions{})
		require.NoError(t, err)

		app, err := handler.resolveAppSelectorName(job.Name, namespace, "job")
		require.NoError(t, err)
		require.Equal(t, "testsuite", app)
	})
}
