package controller

import (
	"context"
	"fmt"
	"time"

	aiv1 "github.com/re-cinq/ai-operator/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *JobReconciler) createJob(ctx context.Context, aiJob aiv1.Job) error {
	logger := log.FromContext(ctx)

	existingJob := &batchv1.Job{}
	err := r.Get(ctx, client.ObjectKey{Name: aiJob.Name, Namespace: aiJob.Namespace}, existingJob)
	if err == nil {
		if err := r.deleteJob(ctx, aiJob); err != nil {
			logger.Error(err, "unable to delete existing job")
			return err
		}
	} else if !apierrors.IsNotFound(err) {
		return err
	}

	// We are mounting the same volume in both init and main containers
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      jobDefaultVolumeName,
			MountPath: "/tmp",
		},
	}

	// Set the environment variable for the Hugging Face token
	huggingFaceTokenVar := corev1.EnvVar{
		Name: "HF_TOKEN",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: huggingFaceSecretName,
				},
				Key: "token",
			},
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      aiJob.Name,
			Namespace: aiJob.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name": aiJob.Name,
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": aiJob.Name,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					InitContainers: []corev1.Container{
						{
							Name:  fmt.Sprintf("%s-init", aiJob.Name),
							Image: aiJob.Spec.Image,
							TTY:   true,
							Command: []string{
								"tune",
								"download",
								aiJob.Spec.Model,
							},
							Env: []corev1.EnvVar{
								huggingFaceTokenVar,
								{
									Name:  "PYTHONUNBUFFERED",
									Value: "1",
								},
							},
							VolumeMounts: volumeMounts,
						},
					},
					Containers: []corev1.Container{
						{
							Name:    aiJob.Name,
							Image:   aiJob.Spec.Image,
							TTY:     true,
							Command: aiJob.Spec.Command,
							Env: []corev1.EnvVar{
								huggingFaceTokenVar,
							},
							VolumeMounts: volumeMounts,
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: jobDefaultVolumeName,
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: aiJob.Name,
								},
							},
						},
					},
				},
			},
		},
	}

	// Set the owner reference to the AI Job
	if err := r.setOwnerReference(&aiJob, job); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	if err := r.Create(ctx, job); err != nil {
		logger.Error(err, "unable to create job")
		return err
	}
	return nil
}

func (r *JobReconciler) deleteJob(ctx context.Context, aiJob aiv1.Job) error {
	logger := log.FromContext(ctx)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      aiJob.Name,
			Namespace: aiJob.Namespace,
		},
	}

	if err := r.Delete(ctx, job); err != nil {
		if !apierrors.IsNotFound(err) {
			logger.Error(err, "unable to delete job")
			return err
		}
		return nil
	}

	return r.waitForJobDeletion(ctx, aiJob)
}

func (r *JobReconciler) waitForJobDeletion(ctx context.Context, aiJob aiv1.Job) error {
	for {
		job := &batchv1.Job{}
		err := r.Get(ctx, client.ObjectKey{Name: aiJob.Name, Namespace: aiJob.Namespace}, job)
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		time.Sleep(time.Second)
	}
}
