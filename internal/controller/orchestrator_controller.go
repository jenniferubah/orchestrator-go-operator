/*
Copyright 2024.

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
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"time"

	//appsv1 "k8s.io/api/apps/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	olmclientset "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"

	orchestratorv1alpha1 "github.com/parodos-dev/orchestrator-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Definition to manage Orchestrator condition status.
const (
	TypeAvailable string = "Available"
	//TypeProgressing string = "Progressing"
	//TypeDegraded    string = "Degraded"
)

// OrchestratorReconciler reconciles a Orchestrator object
type OrchestratorReconciler struct {
	client.Client
	OLMClient olmclientset.Interface
	ClientSet *kubernetes.Clientset
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
}

//+kubebuilder:rbac:groups=orchestrator.parodos.dev,resources=orchestrators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=orchestrator.parodos.dev,resources=orchestrators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=orchestrator.parodos.dev,resources=orchestrators/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=namespaces;events,verbs=list;get;create;delete;patch;watch
//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch
//+kubebuilder:rbac:groups=operators.coreos.com,resources=subscriptions,verbs=get;list;watch;create;delete;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// The Reconcile function compares the state specified by
// the Orchestrator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *OrchestratorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Starting reconciliation")

	// Fetch the Orchestrator instance
	// The purpose is to check if the Custom Resource for the Kind Orchestrator
	// is applied on the cluster if not we return nil to stop the reconciliation
	orchestrator := &orchestratorv1alpha1.Orchestrator{}

	err := r.Get(ctx, req.NamespacedName, orchestrator) // Lookup the Orchestrator instance for this reconcile request
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			logger.Info("orchestrator resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get orchestrator")
		return ctrl.Result{}, err
	}

	// Set the status to Unknown when no status is available - usually initial reconciliation.
	if orchestrator.Status.Conditions == nil || len(orchestrator.Status.Conditions) == 0 {
		meta.SetStatusCondition(
			&orchestrator.Status.Conditions,
			metav1.Condition{
				Type:    TypeAvailable,
				Status:  metav1.ConditionUnknown,
				Reason:  "Reconciling",
				Message: "Starting Reconciliation",
			},
		)
		if err = r.Status().Update(ctx, orchestrator); err != nil {
			logger.Error(err, "Failed to update Orchestrator status")
			return ctrl.Result{}, err
		}
		// Re-fetch orchestrator Custom Resource after updating the status
		if err := r.Get(ctx, req.NamespacedName, orchestrator); err != nil {
			logger.Error(err, "Failed to re fetch orchestrator")
			return ctrl.Result{}, err
		}
	}

	// sonataFlowOperator
	// for creating and deleting use case
	sonataFlowOperator := orchestrator.Spec.SonataFlowOperator
	subscriptionName := sonataFlowOperator.Subscription.Name
	namespace := sonataFlowOperator.Subscription.Namespace

	// check if the sonataflow subscription operator is enabled
	// subscription is disabled,
	if !sonataFlowOperator.Enabled {
		// check if subscription exists using olm client
		sonataFlowSubscription, err := r.OLMClient.OperatorsV1alpha1().Subscriptions(namespace).Get(ctx, subscriptionName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				logger.Info("Subscription resource not found.", "SubscriptionName", subscriptionName, "Namespace", namespace)
			}
			logger.Error(err, "Failed to check Subscription does not exists", "SubscriptionName", subscriptionName)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
		logger.Info("Subscription exists: %s", sonataFlowSubscription.Name)

		// deleting subscription resource
		err = r.OLMClient.OperatorsV1alpha1().Subscriptions(namespace).Delete(ctx, subscriptionName, metav1.DeleteOptions{})
		if err != nil {
			logger.Error(err, "Error occurred while deleting Subscription", "SubscriptionName", subscriptionName, "Namespace", namespace)
			return ctrl.Result{RequeueAfter: 5 * time.Minute}, err
		}
		logger.Info("Successfully deleted Subscription: %s", subscriptionName)
	}

	// Subscription is enabled
	// check if CRD exists;
	sonataCRD := &apiextensionsv1.CustomResourceDefinition{}
	err = r.Get(ctx, types.NamespacedName{Name: subscriptionName, Namespace: namespace}, sonataCRD)
	//subscriptionName = "sonataflowclusterplatforms.sonataflow.org"
	//err = r.Get(ctx, types.NamespacedName{Name: subscriptionName}, sonataCRD)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// CRD does not exist
			logger.Info("CRD resource not found.", "SubscriptionName", subscriptionName, "Namesapce", namespace)

			// install operator via subscription
			logger.Info("Starting subscription installation for", "SubscriptionName", subscriptionName)
			logger.Info("Creating namespace", "Namespace", namespace)

			serverlessLogicNamespace := &corev1.Namespace{}
			// check if namespace exists
			err = r.Get(ctx, types.NamespacedName{Name: namespace}, serverlessLogicNamespace)
			if err != nil {
				if apierrors.IsNotFound(err) {
					// create new namespace
					newNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
					err = r.Create(ctx, newNamespace)
					if err != nil {
						logger.Error(err, "Error occurred when creating namespace", "Namespace", namespace)
						return ctrl.Result{RequeueAfter: 1 * time.Second}, err
					}
				}
				logger.Error(err, "Error occurred when checking namespace exists", "Namespace", namespace)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, err
			}
			subscriptionObject := createSubscriptionObject(subscriptionName, namespace, sonataFlowOperator)
			installedSubscription, err := r.OLMClient.OperatorsV1alpha1().Subscriptions(namespace).Create(ctx, subscriptionObject, metav1.CreateOptions{})

			if err != nil {
				logger.Error(err, "Error occurred while creating Subscription", "SubscriptionName", subscriptionName)
				return ctrl.Result{RequeueAfter: 5 * time.Second}, err
			}

			logger.Info("Successfully installed Operator Subscription", "SubscriptionName", installedSubscription.Name)

		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get CRD resource", "SubscriptionName", subscriptionName)
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, err
	}

	// CRD exists; check if CR exists,
	//sfcCR := &orchestratorv1alpha1.SonataFlowCluster{}
	//err = r.Get(ctx, req.NamespacedName, sfcCR)
	//if err != nil {
	//	if apierrors.IsNotFound(err) {
	//		logger.Info("Sonataflow cluster Resource not found")
	//		// if CR does not exist, create CR
	//		// Create sonataflow cluster CR object
	//		sonataFlowClusterCR := &orchestratorv1alpha1.SonataFlowCluster{
	//			TypeMeta: metav1.TypeMeta{
	//				APIVersion: "sonataflow.org/v1alpha08",  // Replace with your CRD group and version
	//				Kind:       "SonataFlowClusterPlatform", // Replace with your CRD kind
	//			},
	//			ObjectMeta: metav1.ObjectMeta{
	//				Name:      "cluster-platform", // Name of the CR
	//				Namespace: "sonataflow-infra", // Namespace of the CR
	//			},
	//			Spec: orchestratorv1alpha1.SonataFlowClusterSpec{
	//				PlatformRef: orchestratorv1alpha1.PlatformRef{
	//					PlatformName:      "sonataflow-platform",
	//					PlatformNamespace: "sonataflow-infra",
	//				},
	//			},
	//		}
	//
	//		// Create sonataflow cluster CR
	//		if err := r.Create(ctx, sonataFlowClusterCR); err != nil {
	//			logger.Error(err, "Failed to create Custom Resource: %v", sonataFlowClusterCR.Name)
	//			return ctrl.Result{}, err
	//		}
	//		logger.Info("Successfully created SonataFlow Cluster resource %s", sonataFlowClusterCR.Name)
	//		return ctrl.Result{}, nil
	//	}
	//}
	//
	//sfpCR := &orchestratorv1alpha1.SonataFlow{}
	//err = r.Get(ctx, req.NamespacedName, sfpCR)
	//if err != nil {
	//	if apierrors.IsNotFound(err) {
	//		logger.Info("Sonataflow platform not found")
	//		// Create sonataflow platform CR object
	//		sonataFlowPlatformCR := &orchestratorv1alpha1.SonataFlow{
	//			TypeMeta: metav1.TypeMeta{
	//				APIVersion: "sonataflow.org/v1alpha08", // Replace with your CRD group and version
	//				Kind:       "SonataFlowPlatform",       // Replace with your CRD kind
	//			},
	//			ObjectMeta: metav1.ObjectMeta{
	//				Name:      "sonataflow-platform", // Name of the CR
	//				Namespace: "sonataflow-infra",    // Namespace of the CR
	//			},
	//			Spec: orchestratorv1alpha1.SonataFlowPlatformSpec{
	//				Build: orchestratorv1alpha1.PlatformBuild{
	//					Template: orchestratorv1alpha1.Template{Resource: orchestratorv1alpha1.Resource{
	//						Requests: orchestratorv1alpha1.MemoryCpu{
	//							Memory: orchestrator.Spec.OrchestratorPlatform.SonataFlowPlatform.Resources.Requests.Memory,
	//							Cpu:    orchestrator.Spec.OrchestratorPlatform.SonataFlowPlatform.Resources.Requests.Cpu,
	//						},
	//						Limits: orchestratorv1alpha1.MemoryCpu{
	//							Memory: orchestrator.Spec.OrchestratorPlatform.SonataFlowPlatform.Resources.Limits.Memory,
	//							Cpu:    orchestrator.Spec.OrchestratorPlatform.SonataFlowPlatform.Resources.Limits.Cpu,
	//						},
	//					},
	//					},
	//				},
	//				Services: orchestratorv1alpha1.PlatformServices{
	//					DataIndex: orchestratorv1alpha1.DataIndex{
	//						Enabled:     true,
	//						Persistence: getSonataFlowPersistence(orchestrator),
	//					},
	//					JobService: orchestratorv1alpha1.JobService{
	//						Enabled:     true,
	//						Persistence: getSonataFlowPersistence(orchestrator),
	//						//PodTemplate: orchestratorv1alpha1.PodTemplate{},
	//					},
	//				},
	//			},
	//		}
	//		// Create sonataflow platform CR
	//		if err := r.Create(ctx, sonataFlowPlatformCR); err != nil {
	//			logger.Error(err, "Failed to create Custom Resource: %v", sonataFlowPlatformCR.Name)
	//			return ctrl.Result{}, err
	//		}
	//	}
	//}

	// if CR exists, check desired state is the same as current state.

	return ctrl.Result{}, nil
}

//func getSonataFlowPersistence(orchestrator *orchestratorv1alpha1.Orchestrator) orchestratorv1alpha1.Persistence {
//	return orchestratorv1alpha1.Persistence{
//		Postgresql: orchestratorv1alpha1.Postgresql{
//			SecretRef: orchestratorv1alpha1.PostgresAuthSecret{
//				SecretName:  orchestrator.Spec.PostgresDB.AuthSecret.SecretName,
//				UserKey:     orchestrator.Spec.PostgresDB.AuthSecret.UserKey,
//				PasswordKey: orchestrator.Spec.PostgresDB.AuthSecret.PasswordKey,
//			},
//			ServiceRef: orchestratorv1alpha1.ServiceRef{
//				Name:      orchestrator.Spec.PostgresDB.ServiceName,
//				Namespace: orchestrator.Spec.PostgresDB.ServiceNameSpace,
//			},
//		}}
//}

func createSubscriptionObject(subscriptionName string, namespace string, sonataFlowOperator orchestratorv1alpha1.SonataFlowOperator) *v1alpha1.Subscription {
	logger := log.Log.WithName("subscriptionObject")
	logger.Info("Creating subscription object")

	sonataFlowSubscriptionDetails := sonataFlowOperator.Subscription
	subscriptionObject := &v1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: subscriptionName},
		Spec: &v1alpha1.SubscriptionSpec{
			Channel:                sonataFlowSubscriptionDetails.Channel,
			InstallPlanApproval:    v1alpha1.Approval(sonataFlowSubscriptionDetails.InstallPlanApproval),
			CatalogSource:          sonataFlowSubscriptionDetails.SourceName,
			StartingCSV:            sonataFlowSubscriptionDetails.StartingCSV,
			CatalogSourceNamespace: sonataFlowSubscriptionDetails.Namespace,
			Package:                sonataFlowSubscriptionDetails.Name,
		},
	}
	return subscriptionObject
}

// SetupWithManager sets up the controller with the Manager.
func (r *OrchestratorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	//config, err := rest.InClusterConfig()
	config := mgr.GetConfig()

	// Create the OLM clientset using the config
	olmClient, err := olmclientset.NewForConfig(config)
	if err != nil {
		return nil
	}
	r.OLMClient = olmClient

	return ctrl.NewControllerManagedBy(mgr).
		For(&orchestratorv1alpha1.Orchestrator{}).
		//Owns(&orchestratorv1alpha1.SonataFlow{}).
		//Owns(&orchestratorv1alpha1.SonataFlowCluster{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		Complete(r)
}
