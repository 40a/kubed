package framework

import (
	"strings"

	. "github.com/onsi/gomega"
	"k8s.io/api/apps/v1beta1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func int32Ptr(i int32) *int32 { return &i }

func (f *Invocation) NewDeployment(
	name, namespace string,
	labels map[string]string,
	containers []core.Container) *v1beta1.Deployment {
	return &v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				"docker.com/hi-hello": "hello",
			},
			Labels: labels,
		},
		Spec: v1beta1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: core.PodSpec{
					Containers: containers,
					ImagePullSecrets: []core.LocalObjectReference{
						{
							Name: name,
						},
					},
				},
			},
		},
	}
}

func (f *Invocation) NewSecret(name, namespace string, labels map[string]string) *core.Secret {
	return &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		StringData: map[string]string{
			"config.json": `{
  "https://registry-1.docker.io/": {
    "username": "shudipta",
    "password": "pi-shudipta"
  }
}`,
		},
	}
}

func (f *Invocation) AnnotaionsWhoseKeyHasPrefix(deployName, namespace, prefix string) map[string]string {
	res := map[string]string{}
	deploy, err := f.KubeClient.AppsV1beta1().Deployments(namespace).Get(deployName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	annotations := deploy.ObjectMeta.Annotations
	if annotations == nil {
		return res
	}

	for key, val := range annotations {
		if strings.HasPrefix(key, prefix) {
			res[key] = val
		}
	}

	return res
}

func (f *Invocation) DeleteAllDeployments() {
	deployments, err := f.KubeClient.AppsV1beta1().Deployments(metav1.NamespaceAll).List(metav1.ListOptions{
		LabelSelector: labels.Set{
			"app": f.App(),
		}.String(),
	})
	Expect(err).NotTo(HaveOccurred())

	for _, value := range deployments.Items {
		err := f.KubeClient.AppsV1beta1().Deployments(value.Namespace).Delete(value.Name, &metav1.DeleteOptions{})
		if kerr.IsNotFound(err) {
			err = nil
		}
		Expect(err).NotTo(HaveOccurred())
	}
}
