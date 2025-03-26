package controller

import (
	"context"
	"fmt"
	"time"

	aiv1 "github.com/re-cinq/ai-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// Hugging face secret name
	huggingFaceSecretName = "hf-token"
)

func (r *JobReconciler) createSecret(ctx context.Context, aiJob aiv1.Job) (bool, error) {
	logger := log.FromContext(ctx)

	// Construct the secret name
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      huggingFaceSecretName,
			Namespace: aiJob.Namespace,
		},
	}

	if err := r.setOwnerReference(&aiJob, secret); err != nil {
		return false, fmt.Errorf("failed to set owner reference: %w", err)
	}

	// Load the secret to check if it exists
	err := r.Get(ctx, client.ObjectKeyFromObject(secret), secret)

	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create new secret
			secret.Data = map[string][]byte{
				"token": []byte(aiJob.Spec.HuggingFaceSecret),
			}
			// Add the data
			if err := r.Create(ctx, secret); err != nil {
				logger.Error(err, "unable to create secret")
				return false, err
			}
			// Signal that the secret was created, so we need to create the job
			return true, nil
		}
		return false, err
	}

	// Secret exists, update it
	// Check if secret data has changed
	secretUpdated := false
	currentToken := string(secret.Data["token"])
	if currentToken != aiJob.Spec.HuggingFaceSecret {
		secret.StringData = map[string]string{
			"token": aiJob.Spec.HuggingFaceSecret,
		}
		if err := r.Update(ctx, secret); err != nil {
			logger.Error(err, "unable to update secret")
			return false, err
		}
		secretUpdated = true
	}

	return secretUpdated, client.IgnoreNotFound(err)
}

func (r *JobReconciler) deleteSecret(ctx context.Context, aiJob aiv1.Job) error {
	logger := log.FromContext(ctx)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      huggingFaceSecretName,
			Namespace: aiJob.Namespace,
		},
	}

	if err := r.Delete(ctx, secret); err != nil {
		if !apierrors.IsNotFound(err) {
			logger.Error(err, "unable to delete secret")
			return err
		}
		return nil
	}

	return r.waitForSecretDeletion(ctx, aiJob)
}

func (r *JobReconciler) waitForSecretDeletion(ctx context.Context, aiJob aiv1.Job) error {
	for {
		secret := &corev1.Secret{}
		err := r.Get(ctx, client.ObjectKey{Name: huggingFaceSecretName, Namespace: aiJob.Namespace}, secret)
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		time.Sleep(time.Second)
	}
}
