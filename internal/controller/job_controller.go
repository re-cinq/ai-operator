/*
Copyright 2025.

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

package controller

import (
	"context"
	aiv1 "github.com/re-cinq/ai-operator/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"slices"
	"time"
)

const (
	jobFinalizerName     = "job.ai.re-cinq.com/finalizer"
	jobDefaultVolumeName = "model"
)

// JobReconciler reconciles a Job object
type JobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=ai.re-cinq.com,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ai.re-cinq.com,resources=jobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ai.re-cinq.com,resources=jobs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Job object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *JobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling AI Job %s in %s", req.Name, req.NamespacedName)

	// Load the AI Job specs
	var aiJob aiv1.Job
	if err := r.Get(ctx, req.NamespacedName, &aiJob); err != nil {
		logger.Error(err, "unable to fetch AI Job")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Make sure we have a valid spec
	if err := aiJob.Spec.Validate(); err != nil {
		logger.Error(err, "invalid job spec")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the AI Job is marked for deletion
	if !aiJob.ObjectMeta.DeletionTimestamp.IsZero() {
		if err := r.delete(ctx, aiJob); err != nil {
			logger.Error(err, "failed to delete resources")
			return ctrl.Result{RequeueAfter: time.Second * 15}, err
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if it doesn't exist
	if !slices.Contains(aiJob.Finalizers, jobFinalizerName) {
		aiJob.Finalizers = append(aiJob.Finalizers, jobFinalizerName)
		if err := r.Update(ctx, &aiJob); err != nil {
			logger.Error(err, "failed to add finalizer")
			return ctrl.Result{RequeueAfter: time.Second * 5}, err
		}
		return ctrl.Result{}, nil
	}

	// Handle creation/update
	if err := r.create(ctx, aiJob); err != nil {
		logger.Error(err, "failed to reconcile resources")
		return ctrl.Result{RequeueAfter: time.Second * 15}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *JobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aiv1.Job{}).
		Named("job").
		Complete(r)
}

func (r *JobReconciler) needsUpdate(aiJob aiv1.Job, existingJob *batchv1.Job) bool {
	// Check if the image has changed
	if len(existingJob.Spec.Template.Spec.Containers) > 0 {
		if existingJob.Spec.Template.Spec.Containers[0].Image != aiJob.Spec.Image {
			return true
		}

		// Check if the command has changed
		if !slices.Equal(existingJob.Spec.Template.Spec.Containers[0].Command, aiJob.Spec.Command) {
			return true
		}
	}

	// Check if volumes have changed
	for _, volume := range existingJob.Spec.Template.Spec.Volumes {
		if volume.Name == jobDefaultVolumeName {
			if volume.PersistentVolumeClaim.ClaimName != aiJob.Name {
				return true
			}
		}
	}

	return false
}

func (r *JobReconciler) setOwnerReference(aiJob *aiv1.Job, obj client.Object) error {
	return ctrl.SetControllerReference(aiJob, obj, r.Scheme)
}

func (r *JobReconciler) updateStatus(ctx context.Context, aiJob *aiv1.Job, state, details string) error {
	aiJob.Status.State = state
	aiJob.Status.Details = details
	return r.Status().Update(ctx, aiJob)
}

// Delete the AI Job
func (r *JobReconciler) delete(ctx context.Context, aiJob aiv1.Job) error {
	// Delete the Job
	if err := r.deleteJob(ctx, aiJob); err != nil {
		return err
	}

	// Delete the PVC
	if err := r.deletePVC(ctx, aiJob); err != nil {
		return err
	}

	// Delete the Secret
	if err := r.deleteSecret(ctx, aiJob); err != nil {
		return err
	}

	// Remove finalizer after successful deletion
	aiJob.Finalizers = slices.DeleteFunc(aiJob.Finalizers, func(s string) bool {
		return s == jobFinalizerName
	})
	if err := r.Update(ctx, &aiJob); err != nil {
		return err
	}

	return nil
}

// Called when an AI Job is created or updated
func (r *JobReconciler) create(ctx context.Context, aiJob aiv1.Job) error {
	//logger := log.FromContext(ctx)
	secretUpdated, err := r.createSecret(ctx, aiJob)
	if err != nil {
		return err
	}

	pvcUpdated, err := r.createPVC(ctx, aiJob)
	if err != nil {
		return err
	}

	// If secret was updated, recreate the job
	// If secret or PVC was updated, recreate the job
	if secretUpdated || pvcUpdated {
		if err := r.createJob(ctx, aiJob); err != nil {
			return err
		}
	}

	return nil
}
