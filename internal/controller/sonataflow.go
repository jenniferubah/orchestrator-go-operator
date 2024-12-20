//
// Copyright (c) 2024 Red Hat, Inc.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	sonataapi "github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"
	olmclientset "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"
	orchestratorv1alpha1 "github.com/parodos-dev/orchestrator-operator/api/v1alpha1"
	"github.com/parodos-dev/orchestrator-operator/internal/controller/kube"
	"github.com/parodos-dev/orchestrator-operator/internal/controller/util"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	SonataFlowAPIVersion             = "sonataflow.org/v1alpha08"
	SonataFlowPlatformCRName         = "sonataflow-platform"
	SonataFlowNamespace              = "sonataflow-infra"
	SonataFlowPlatformKind           = "SonataFlowPlatform"
	SonataFlowClusterPlatformKind    = "SonataFlowClusterPlatform"
	SonataFlowClusterPlatformCRName  = "cluster-platform"
	SonataFlowClusterPlatformCRDName = "sonataflowclusterplatforms.sonataflow.org"
	SonataFlowSubscriptionName       = "logic-operator-rhel8"
)

func getSonataFlowPersistence(orchestrator *orchestratorv1alpha1.Orchestrator) *sonataapi.PersistenceOptionsSpec {
	return &sonataapi.PersistenceOptionsSpec{
		PostgreSQL: &sonataapi.PersistencePostgreSQL{
			SecretRef: sonataapi.PostgreSQLSecretOptions{
				Name:        orchestrator.Spec.PostgresDB.AuthSecret.SecretName,
				UserKey:     orchestrator.Spec.PostgresDB.AuthSecret.UserKey,
				PasswordKey: orchestrator.Spec.PostgresDB.AuthSecret.PasswordKey,
			},
			ServiceRef: &sonataapi.PostgreSQLServiceOptions{
				SQLServiceOptions: &sonataapi.SQLServiceOptions{
					Name:         orchestrator.Spec.PostgresDB.ServiceName,
					Namespace:    orchestrator.Spec.PostgresDB.ServiceNameSpace,
					DatabaseName: orchestrator.Spec.PostgresDB.DatabaseName,
				},
			},
		},
	}
}

func handleSonataFlowClusterCR(ctx context.Context, client client.Client, crName string) error {
	logger := log.FromContext(ctx)
	// check sonataflowlusterplatform CR exists
	sfcCR := &sonataapi.SonataFlowClusterPlatform{}

	err := client.Get(ctx, types.NamespacedName{Name: crName, Namespace: SonataFlowNamespace}, sfcCR)
	if err == nil {
		// CR exists; check for CR updates
		logger.Info("CR resource  found.", "CR-Name", crName, "Namespace", SonataFlowNamespace)
		sfcCR.Spec = getSonataFlowClusterSpec()
		if err = client.Update(ctx, sfcCR); err != nil {
			logger.Error(err, "Failed to update CR", "CR-Name", sfcCR.Name)
		}
		return nil
	} else {
		if apierrors.IsNotFound(err) {
			// Create sonataflowcluster CR object
			sonataFlowClusterCR := &sonataapi.SonataFlowClusterPlatform{
				TypeMeta: metav1.TypeMeta{
					APIVersion: SonataFlowAPIVersion,
					Kind:       SonataFlowClusterPlatformKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      SonataFlowClusterPlatformCRName,
					Namespace: SonataFlowNamespace,
					Labels:    kube.AddLabel(),
				},
				Spec: getSonataFlowClusterSpec(),
			}

			// Create sonataflow cluster CR
			if err := client.Create(ctx, sonataFlowClusterCR); err != nil {
				logger.Error(err, "Error occurred when creating Custom Resource", "CR-Name", crName)
				return err
			}
			logger.Info("Successfully created SonataFlowClusterPlatform resource %s", sonataFlowClusterCR.Name)
			return nil
		}
		logger.Error(err, "Error occurred when retrieving SonataFlowClusterPlatform CR", "CR-Name", crName)
	}
	return err
}

func getSonataFlowClusterSpec() sonataapi.SonataFlowClusterPlatformSpec {
	return sonataapi.SonataFlowClusterPlatformSpec{
		PlatformRef: sonataapi.SonataFlowPlatformRef{
			Name:      SonataFlowClusterPlatformCRName,
			Namespace: SonataFlowNamespace,
		},
	}
}

func handleSonataFlowPlatformCR(
	ctx context.Context,
	client client.Client,
	orchestrator *orchestratorv1alpha1.Orchestrator,
	crName string) error {
	logger := log.FromContext(ctx)

	logger.Info("Starting CR creation for SonataFlowPlatform...")

	sfpCR := &sonataapi.SonataFlowPlatform{}
	err := client.Get(ctx, types.NamespacedName{
		Namespace: SonataFlowNamespace,
		Name:      SonataFlowPlatformCRName,
	}, sfpCR)

	if err == nil {
		// CR exists; check for CR updates
		logger.Info("CR resource  found.", "CR-Name", crName, "Namespace", SonataFlowNamespace)
		err = client.Update(ctx, sfpCR)

		return nil
	} else {
		if apierrors.IsNotFound(err) {
			logger.Info("SonataFlowPlatform not found. Proceed to creating CR...")
			// Create sonataflow platform CR object

			sonataFlowPlatformCR := &sonataapi.SonataFlowPlatform{
				TypeMeta: metav1.TypeMeta{
					APIVersion: SonataFlowAPIVersion,
					Kind:       SonataFlowPlatformKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      SonataFlowPlatformCRName,
					Namespace: SonataFlowNamespace,
					Labels:    kube.AddLabel(),
				},
				Spec: getSonataFlowPlatformSpec(orchestrator),
			}
			logger.Info("Persistence function", "Persistent", getSonataFlowPersistence(orchestrator))
			// Create sonataflowplatform CR
			if err := client.Create(ctx, sonataFlowPlatformCR); err != nil {
				logger.Error(err, "Failed to create Custom Resource", "CR-Name", crName)
				return err
			}
			logger.Info("Successfully created CR", "CR-Name", sonataFlowPlatformCR.Name)
			return nil
		}
		logger.Error(err, "Error occurred when retrieving SonataFlowPlatform CR", "CR-Name", crName)
	}
	return err
}

func getSonataFlowPlatformSpec(orchestrator *orchestratorv1alpha1.Orchestrator) sonataapi.SonataFlowPlatformSpec {
	limitResourceMap := make(map[corev1.ResourceName]resource.Quantity)

	cpuQuantity, _ := resource.ParseQuantity(orchestrator.Spec.OrchestratorPlatform.SonataFlowPlatform.Resources.Limits.Cpu)
	memoryQuantity, _ := resource.ParseQuantity(orchestrator.Spec.OrchestratorPlatform.SonataFlowPlatform.Resources.Limits.Memory)
	limitResourceMap[corev1.ResourceCPU] = cpuQuantity
	limitResourceMap[corev1.ResourceMemory] = memoryQuantity
	//logger.Info("Limit Map", "Map", limitResourceMap)

	requestResourceMap := make(map[corev1.ResourceName]resource.Quantity)
	requestCpuQuantity, _ := resource.ParseQuantity(orchestrator.Spec.OrchestratorPlatform.SonataFlowPlatform.Resources.Requests.Cpu)
	requestMemoryQuantity, _ := resource.ParseQuantity(orchestrator.Spec.OrchestratorPlatform.SonataFlowPlatform.Resources.Requests.Memory)
	requestResourceMap[corev1.ResourceCPU] = requestCpuQuantity
	requestResourceMap[corev1.ResourceMemory] = requestMemoryQuantity
	//logger.Info("Request Map", "Map", requestResourceMap)

	return sonataapi.SonataFlowPlatformSpec{
		Build: sonataapi.BuildPlatformSpec{
			Template: sonataapi.BuildTemplate{
				Resources: corev1.ResourceRequirements{
					Limits:   limitResourceMap,
					Requests: requestResourceMap,
				},
			}},
		Services: &sonataapi.ServicesPlatformSpec{
			DataIndex: &sonataapi.ServiceSpec{
				Enabled:     util.MakePointer(true),
				Persistence: getSonataFlowPersistence(orchestrator),
			},
			JobService: &sonataapi.ServiceSpec{
				Enabled:     util.MakePointer(true),
				Persistence: getSonataFlowPersistence(orchestrator),
			},
		},
	}
}

func handleSonataFlowCleanUp(ctx context.Context, client client.Client, olmClientSet olmclientset.Clientset) error {
	logger := log.FromContext(ctx)
	// remove all namespace
	if err := kube.CleanUpNamespace(ctx, SonataFlowNamespace, client); err != nil {
		logger.Error(err, "Error occurred when deleting namespace", "NS", KnativeEventingNamespacedName)
		return err
	}
	if err := kube.CleanUpSubscriptionAndCSV(ctx, olmClientSet, SonataFlowSubscriptionName, SonataFlowNamespace); err != nil {
		logger.Error(err, "Error occurred when deleting Subscription and CSV", "Subscription", SonataFlowSubscriptionName)
		return err
	}
	// remove all CRDs, optional (ensure all CRs and namespace have been removed first)
	return nil
}
