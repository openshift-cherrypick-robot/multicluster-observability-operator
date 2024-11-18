// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project
// Licensed under the Apache License 2.0

package rendering

import (
	"context"
	"os"
	"path"
	"testing"

	imagev1 "github.com/openshift/api/image/v1"
	fakeimageclient "github.com/openshift/client-go/image/clientset/versioned/fake"
	fakeimagev1client "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1/fake"
	"github.com/stolostron/multicluster-observability-operator/operators/multiclusterobservability/pkg/config"
	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	mcoshared "github.com/stolostron/multicluster-observability-operator/operators/multiclusterobservability/api/shared"
	mcov1beta2 "github.com/stolostron/multicluster-observability-operator/operators/multiclusterobservability/api/v1beta2"
	templatesutil "github.com/stolostron/multicluster-observability-operator/operators/pkg/rendering/templates"
)

func TestRender(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir %v", err)
	}
	templatesPath := path.Join(path.Dir(path.Dir(wd)), "manifests")
	os.Setenv(templatesutil.TemplatesPathEnvVar, templatesPath)
	defer os.Unsetenv(templatesutil.TemplatesPathEnvVar)

	mchcr := &mcov1beta2.MultiClusterObservability{
		TypeMeta:   metav1.TypeMeta{Kind: "MultiClusterObservability"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "test"},
		Spec: mcov1beta2.MultiClusterObservabilitySpec{
			ImagePullPolicy: "IfNotPresent",
			ImagePullSecret: "test",
			StorageConfig: &mcov1beta2.StorageConfig{
				MetricObjectStorage: &mcoshared.PreConfiguredStorage{
					Key:  "test",
					Name: "test",
				},
				StorageClass:            "gp2",
				AlertmanagerStorageSize: "1Gi",
				CompactStorageSize:      "1Gi",
				RuleStorageSize:         "1Gi",
				ReceiveStorageSize:      "1Gi",
				StoreStorageSize:        "1Gi",
			},
		},
	}

	clientCa := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "extension-apiserver-authentication",
			Namespace: "kube-system",
		},
		Data: map[string]string{
			"client-ca-file": "test",
		},
	}
	kubeClient := fake.NewClientBuilder().WithObjects(clientCa).Build()

	imageClient := &fakeimagev1client.FakeImageV1{Fake: &(fakeimageclient.NewSimpleClientset().Fake)}
	_, err = imageClient.ImageStreams(config.OauthProxyImageStreamNamespace).Create(context.Background(),
		&imagev1.ImageStream{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.OauthProxyImageStreamName,
				Namespace: config.OauthProxyImageStreamNamespace,
			},
			Spec: imagev1.ImageStreamSpec{
				Tags: []imagev1.TagReference{
					{
						Name: "v4.4",
						From: &corev1.ObjectReference{
							Kind: "DockerImage",
							Name: "quay.io/openshift-release-dev/ocp-v4.0-art-dev",
						},
					},
				},
			},
		}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}

	renderer := NewMCORenderer(mchcr, kubeClient, imageClient)
	_, err = renderer.Render()
	if err != nil {
		t.Fatalf("failed to render MultiClusterObservability: %v", err)
	}
}

func TestGetOauthProxyFromImageStreams(t *testing.T) {
	imageClient := &fakeimagev1client.FakeImageV1{Fake: &(fakeimageclient.NewSimpleClientset().Fake)}
	_, err := imageClient.ImageStreams(config.OauthProxyImageStreamNamespace).Create(context.Background(),
		&imagev1.ImageStream{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.OauthProxyImageStreamName,
				Namespace: config.OauthProxyImageStreamNamespace,
			},
			Spec: imagev1.ImageStreamSpec{
				Tags: []imagev1.TagReference{
					{
						Name: "v4.4",
						From: &corev1.ObjectReference{
							Kind: "DockerImage",
							Name: "quay.io/openshift-release-dev/ocp-v4.0-art-dev",
						},
					},
				},
			},
		}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	found, oauthProxyImage := config.GetOauthProxyImage(imageClient)
	if !found {
		t.Fatal("Failed to get oauth proxy image")
	}
	assert.Equal(t, "quay.io/openshift-release-dev/ocp-v4.0-art-dev", oauthProxyImage)
}
