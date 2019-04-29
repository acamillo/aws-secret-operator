package awssecret

import (
	"context"
	"encoding/json"
	"fmt"

	acamillov1alpha1 "github.com/acamillo/aws-secret-operator/pkg/apis/acamillo/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

var log = logf.Log.WithName("controller_awssecret")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new AWSSecret Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileAWSSecret{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("awssecret-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource AWSSecret
	err = c.Watch(&source.Kind{Type: &acamillov1alpha1.AWSSecret{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileAWSSecret{}

// ReconcileAWSSecret reconciles a AWSSecret object
type ReconcileAWSSecret struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a AWSSecret object and makes changes based on the state read
// and what is in the AWSSecret.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileAWSSecret) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling AWSSecret")

	// Fetch the AWSSecret instance
	instance := &acamillov1alpha1.AWSSecret{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Fetch the secret from AWS Secret Manager and creates an equivalent k8s Secret resource
	secret, err := r.newSecretForCR(instance)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to retrieve the secret from AWS %s: %v", request.Name, err)
	}

	// Set AWSSecret instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, secret, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Secret already exists
	found := &corev1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		err = r.client.Create(context.TODO(), secret)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Secret created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	reqLogger.Info("Updating the Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
	err = r.client.Update(context.TODO(), secret)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Secret updated successfully - don't requeue
	return reconcile.Result{}, nil
}

// newSecretForCR returns a Secret resource with the same name/namespace as the cr
func (r *ReconcileAWSSecret) newSecretForCR(cr *acamillov1alpha1.AWSSecret) (*corev1.Secret, error) {

	// create a session and use it for retrieving a secret from AWS' Secret Manager
	s := session.Must(session.NewSession())
	sm := secretsmanager.New(s)

	ref := cr.Spec.SecretsManagerRef
	getSecInput := &secretsmanager.GetSecretValueInput{
		SecretId: &ref.SecretId,
		VersionId: &ref.VersionId,
	}

	output, err := sm.GetSecretValue(getSecInput)
	if err != nil {
		return nil, err
	}

	data := map[string]string{}
	if err := json.Unmarshal([]byte(*output.SecretString), &data); err != nil {
		return nil, fmt.Errorf("failed to get json secret as map")
	}

	labels := map[string]string{
		//"app": cr.Name,
		"app": "aws-secret-operator",
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		StringData: data,
	}, nil
}
