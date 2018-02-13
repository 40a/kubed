package e2e

import (
	"github.com/appscode/kubed/test/framework"
	"github.com/appscode/kutil/meta"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/apps/v1beta1"
	core "k8s.io/api/core/v1"
	"time"

	"github.com/appscode/go/log"
)

var _ = FDescribe("Extract Docker Label", func() {
	var (
		f                  *framework.Invocation
		secret             *core.Secret
		deployment, deploy *v1beta1.Deployment
		labels             map[string]string
		name, namespace    string
		containers         []core.Container
	)

	BeforeEach(func() {
		f = root.Invoke()
		name = f.App()
		namespace = "kube-system"
		labels = map[string]string{
			"app": f.App(),
		}

		secret = f.NewSecret(name, namespace, labels)
	})

	JustBeforeEach(func() {
		By("Creating secret")
		_, err := root.KubeClient.CoreV1().Secrets(secret.Namespace).Create(secret)
		Expect(err).NotTo(HaveOccurred())
		//f.EventuallyNumOfSecrets(f.Namespace()).Should(BeNumerically("==", 1))

		By("Creating deployment")
		deploy, err = root.KubeClient.AppsV1beta1().Deployments(deployment.Namespace).Create(deployment)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		f.DeleteAllSecrets()
		f.DeleteAllDeployments()
	})

	FDescribe("Adding Annotaions", func() {
		FContext("When docker image contains labels", func() {
			BeforeEach(func() {
				containers = []core.Container{
					{
						Name:  "labels",
						Image: "shudipta/labels",
						Ports: []core.ContainerPort{
							{
								ContainerPort: 80,
							},
						},
					},
				}
				deployment = f.NewDeployment(name, namespace, labels, containers)
			})

			FIt("Should add annotations", func() {
				imageAnnotations := map[string]string{
					//"docker.com/hi-hello": "hello",
					"docker.com/labels-git-commit": "unkown",
				}
				time.Sleep(time.Second * 10)
				By("\"git-commit\" annotations should be present")
				annotations := f.AnnotaionsWhoseKeyHasPrefix(deploy.Name, deploy.Namespace, "docker.com/")
				log.Infoln("annotations =", annotations)
				Expect(annotations).To(Equal(imageAnnotations))

				By("\"hello\" annotation shouldn't be present")
				value, err := meta.GetStringValue(annotations, "docker.com/hi-hello")
				Expect(err).To(HaveOccurred())
				Expect(value).To(Equal(""))
			})
		})

		FContext("When docker image doesn't contain any labels", func() {
			BeforeEach(func() {
				containers = []core.Container{
					{
						Name:  "book-server",
						Image: "shudipta/book_server:v1",
						Ports: []core.ContainerPort{
							{
								ContainerPort: 10000,
							},
						},
					},
				}
				deployment = f.NewDeployment(name, namespace, labels, containers)
			})

			FIt("Shouldn't add annotations", func() {
				imageAnnotations := map[string]string{}
				time.Sleep(time.Second * 10)
				By("\"git-commit\" annotations should be present")
				annotations := f.AnnotaionsWhoseKeyHasPrefix(deploy.Name, deploy.Namespace, "docker.com/")
				log.Infoln("annotations =", annotations)
				Expect(annotations).To(Equal(imageAnnotations))
			})
		})

		FContext("When docker images aren't found with the provided secrets", func() {
			BeforeEach(func() {
				containers = []core.Container{
					{
						Name:  "guard",
						Image: "nightfury1204/guard:azure",
						Ports: []core.ContainerPort{
							{
								ContainerPort: 10000,
							},
						},
					},
				}
				deployment = f.NewDeployment(name, namespace, labels, containers)
			})

			FIt("Shouldn't found images", func() {
				imageAnnotations := map[string]string{}
				time.Sleep(time.Second * 10)
				By("\"git-commit\" annotations should be present")
				annotations := f.AnnotaionsWhoseKeyHasPrefix(deploy.Name, deploy.Namespace, "docker.com/")
				Expect(annotations).To(Equal(imageAnnotations))
			})
		})
	})
})
