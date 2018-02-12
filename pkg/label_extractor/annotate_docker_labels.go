package label_extractor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/appscode/go/log"
	apps_util "github.com/appscode/kutil/apps/v1beta1"
	core_util "github.com/appscode/kutil/core/v1"
	"github.com/docker/docker/api/types"
	"github.com/heroku/docker-registry-client/registry"
	"k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/util/parsers"
)

// getAllSecrets() takes imagePullSecrets and return the list of secret names as an array of
// string
func (l *LabelExtractor) getAllSecrets(imagePullSecrets []corev1.LocalObjectReference) []string {
	secretNames := []string{}
	for _, secretName := range imagePullSecrets {
		secretNames = append(secretNames, secretName.Name)
	}

	return secretNames
}

// This method takes a deployment <deploy> and checks if there exists any labels in container images
// at PodTemplateSpec. If exists then add them to annotation of the <deploy>. It uses the secrets
// provided at 'imagePullSecrets' for getting labels from images
func (l *LabelExtractor) Annotate(deploy *v1beta1.Deployment) error {
	secretNames := l.getAllSecrets(deploy.Spec.Template.Spec.ImagePullSecrets)

	annotations := make(map[string]string)
	for _, cont := range deploy.Spec.Template.Spec.Containers {
		img := cont.Image

		labels, err := l.GetLabels(deploy.ObjectMeta.GetNamespace(), img, secretNames)
		if err != nil {
			return err
		}

		var mergeErr error = nil
		core_util.UpsertMap(annotations, labels)
		if mergeErr != nil {
			return fmt.Errorf("for img %s %v", mergeErr)
		}
	}

	_, status, err := apps_util.PatchDeployment(l.kubeClient, deploy, func(deployment *v1beta1.Deployment) *v1beta1.Deployment {
		deployment.ObjectMeta.Annotations = annotations

		return deployment
	})
	if err != nil {
		return fmt.Errorf("error status = %s: %v", status, err)
	}

	return nil
}

// This method takes namespace_name <namespace> of provided secrets <secretNames> and a docker image
// name <image>. For each secret it reads the config data of secret and store it to registrySecrets
// (map[string]RegistrySecret) where the api url is the key and value is the credentials. Then it tries
// to extract labels of the <image> for all secrets' content. If found then returns labels otherwise
// returns corresponding error. If <image> is not found with the secret info, then it tries with the
// public docker url="https://registry-1.docker.io/"
func (l *LabelExtractor) GetLabels(namespace, image string, secretNames []string) (map[string]string, error) {
	log.Infoln("img =", image, "secret names =", secretNames)

	repo, tag, _, err := parsers.ParseImageName(image)
	if err != nil {
		return nil, err
	}
	repoName := repo[10:]

	for _, item := range secretNames {
		secret, err := l.kubeClient.CoreV1().Secrets(namespace).Get(item, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("couldn't get secret(%s): %v", item, err)
		}

		configData := []byte{}
		for _, val := range secret.Data {
			configData = append(configData, val...)
			break
		}
		log.Infoln("config.json =", string(configData))

		var registrySecrets map[string]RegistrySecret
		err = json.NewDecoder(bytes.NewReader(configData)).Decode(&registrySecrets)
		if err != nil {
			return nil, fmt.Errorf("couldn't decode the configData for secret(%s): %v", item, err)
		}

		for key, val := range registrySecrets {
			labels, extracErr, errStatus := l.ExtractLabelsForThisCred(key, val.Username, val.Password, repoName, tag)

			if errStatus != 0 {
				err = fmt.Errorf("%v\n%v", err, extracErr)
				continue
			}

			return labels, err
		}
	}

	url := "https://registry-1.docker.io/"
	username := "" // anonymous
	pass := ""     // anonymous
	labels, extractErr, errStatus := l.ExtractLabelsForThisCred(url, username, pass, repoName, tag)

	if errStatus != 0 {
		err = fmt.Errorf("%v\n%v", err, extractErr)
		return nil, fmt.Errorf("couldn't find image(%s): %v", image, err)
	}

	return labels, err
}

// This method returns the labels of docker image. The image name is <reopName/tag> and the api url
// is <url>. The essential credentials are <username> and <pass>. If image is found it returns tuple
// {labels, err=nil, status=0}, otherwise it returns tuple {label=nil, err, status}
func (l *LabelExtractor) ExtractLabelsForThisCred(
	url, username, pass string,
	repoName, tag string) (map[string]string, error, int) {

	hub := &registry.Registry{
		URL: url,
		Client: &http.Client{
			Transport: registry.WrapTransport(http.DefaultTransport, url, username, pass),
		},
		Logf: registry.Quiet,
	}

	manifest, err := hub.ManifestV2(repoName, tag)
	if err != nil {
		return nil,
			fmt.Errorf("couldn't get the manifest for credential(url->%s, username->%s, pass->%s): %v",
				url, username, pass, err),
			1
	}

	reader, err := hub.DownloadLayer(repoName, manifest.Config.Digest)
	if err != nil {
		return nil,
			fmt.Errorf("couldn't get encoded imageInspect for credential(url->%s, username->%s, pass->%s): %v",
				url, username, pass, err),
			2
	}

	var cfg types.ImageInspect
	defer reader.Close()
	err = json.NewDecoder(reader).Decode(&cfg)
	if err != nil {
		return nil,
			fmt.Errorf("couldn't get decode imageInspect for credential(url->%s, username->%s, pass->%s): %v",
				url, username, pass, err),
			3
	}

	return cfg.Config.Labels, nil, 0
}
