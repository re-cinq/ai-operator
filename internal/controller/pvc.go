package controller

import (
	"context"
	"fmt"
	"time"

	aiv1 "github.com/re-cinq/ai-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *JobReconciler) createPVC(ctx context.Context, aiJob aiv1.Job) (bool, error) {
	logger := log.FromContext(ctx)

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      aiJob.Name,
			Namespace: aiJob.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name": aiJob.Name,
			},
		},
	}

	if err := r.setOwnerReference(&aiJob, pvc); err != nil {
		return false, fmt.Errorf("failed to set owner reference: %w", err)
	}

	// Default access mode, if not specified
	accessModes := aiJob.Spec.AccessModes
	if len(accessModes) == 0 {
		accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}

	// Default storage class, if not specified
	requestedStorageClassName := aiJob.Spec.StorageClassName
	if requestedStorageClassName == "" {
		requestedStorageClassName = "local-path"
	}

	// Check if PVC exists
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(pvc), pvc)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create new PVC
			pvc.Spec = corev1.PersistentVolumeClaimSpec{
				StorageClassName: &requestedStorageClassName,
				AccessModes:      accessModes,
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: *resource.NewQuantity(int64(aiJob.Spec.DiskSize)*1024*1024*1024, resource.BinarySI),
					},
				},
			}
			if err := r.Client.Create(ctx, pvc); err != nil {
				logger.Error(err, "unable to create PVC")
				return false, err
			}
			// Signal that the PVC was created, so we need to create the job
			return true, nil
		}
		return false, err
	}

	// Check if size has changed
	currentSize := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	requestedSize := resource.NewQuantity(int64(aiJob.Spec.DiskSize)*1024*1024*1024, resource.BinarySI)
	requestedSize.Add(currentSize)

	// Check if the storage class has changed
	currentStorageClassName := *pvc.Spec.StorageClassName

	if currentSize.Cmp(*requestedSize) != 0 || requestedStorageClassName != currentStorageClassName {
		// Delete existing PVC
		if err := r.deletePVC(ctx, aiJob); err != nil {
			return false, err
		}

		// Create new PVC with updated size
		pvc.Spec.Resources.Requests[corev1.ResourceStorage] = *requestedSize
		if err := r.Client.Create(ctx, pvc); err != nil {
			logger.Error(err, "unable to create PVC with new size")
			return false, err
		}
		return true, nil
	}

	return false, nil
}

func (r *JobReconciler) deletePVC(ctx context.Context, aiJob aiv1.Job) error {
	logger := log.FromContext(ctx)
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      aiJob.Name,
			Namespace: aiJob.Namespace,
		},
	}

	if err := r.Client.Delete(ctx, pvc); err != nil {
		if !apierrors.IsNotFound(err) {
			logger.Error(err, "unable to delete PVC")
			return err
		}
		return nil
	}

	return r.waitForPVCDeletion(ctx, aiJob)
}

func (r *JobReconciler) waitForPVCDeletion(ctx context.Context, aiJob aiv1.Job) error {
	// Wait for PVC deletion
	for {
		pvc := &corev1.PersistentVolumeClaim{}
		err := r.Client.Get(ctx, client.ObjectKey{Name: aiJob.Name, Namespace: aiJob.Namespace}, pvc)
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		time.Sleep(time.Second)
	}
}
