package label_extractor

import (
	"sync"

	ac_log "github.com/appscode/go/log"
	"k8s.io/api/apps/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type ExtractDockerLabel struct {
	kubeClient kubernetes.Interface

	enable bool
	lock   sync.RWMutex
}

type RegistrySecret struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func New(kubeClient kubernetes.Interface) *ExtractDockerLabel {
	return &ExtractDockerLabel{
		kubeClient: kubeClient,
	}
}

func (l *ExtractDockerLabel) Configure(enable bool) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.enable = enable
}

func (l *ExtractDockerLabel) ExtractDockerLabelHandler() cache.ResourceEventHandler {
	return l
}

func (l *ExtractDockerLabel) OnAdd(obj interface{}) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	if !l.enable {
		return
	}

	if res, ok := obj.(*v1beta1.Deployment); ok {
		if err := l.Annotate(res); err != nil {
			ac_log.Errorln(err)
		}
	}
}

func (l *ExtractDockerLabel) OnUpdate(oldObj, newObj interface{}) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	if !l.enable {
		return
	}

	oldRes, ok := oldObj.(*v1beta1.Deployment)
	if !ok {
		return
	}
	newRes, ok := newObj.(*v1beta1.Deployment)
	if !ok {
		return
	}

	if oldRes.ResourceVersion == newRes.ResourceVersion {
		return
	} else {
		if err := l.Annotate(newRes); err != nil {
			ac_log.Errorln(err)
		}
	}
}

func (l *ExtractDockerLabel) OnDelete(obj interface{}) {}
