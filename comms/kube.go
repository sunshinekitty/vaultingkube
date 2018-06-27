package comms

import (
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	core_v1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// VKKube is a wrapper around Kube client
type VKKube struct {
	Client *kubernetes.Clientset
	Logger *logrus.Logger
}

// NewVKKubeClient returns a new VKKube client
// Reads from ~/.kube/config and falls back to in-cluster config
func NewVKKubeClient(logger *logrus.Logger) (*VKKube, error) {
	var k *VKKube
	var kubeconfig string
	kubeconfigPath := fmt.Sprintf("%s/.kube/config", os.Getenv("HOME"))

	if _, err := os.Stat(kubeconfigPath); !os.IsNotExist(err) {
		kubeconfig = kubeconfigPath
	}

	// BuildConfigFromFlags will fall back to inClusterConfig if kubeconfig is empty
	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)

	if err != nil {
		panic(err.Error())
	}

	kc, err := kubernetes.NewForConfig(restConfig)

	if err != nil {
		return nil, err
	}

	k = &VKKube{
		Client: kc,
		Logger: logger,
	}

	return k, nil
}

// IsManaged determines if a given cm/secret is managed by VaultingKube
func (k *VKKube) IsManaged(name string, secretType string, namespace string) bool {
	var annotations map[string]string
	if secretType == "configmaps" {
		r, err := k.Client.CoreV1().ConfigMaps(namespace).Get(name, meta_v1.GetOptions{})
		annotations = r.Annotations
		if k8s_errors.IsNotFound(err) {
			k.Logger.Infof("Configmap %s in namespace %s does not appear to exist, assuming management", name, namespace)
			return true
		}
		if err != nil {
			k.Logger.Error(err)
			return false
		}
	} else {
		r, err := k.Client.CoreV1().Secrets(namespace).Get(name, meta_v1.GetOptions{})
		annotations = r.Annotations
		if k8s_errors.IsNotFound(err) {
			k.Logger.Infof("Secret %s in namespace %s does not appear to exist, assuming management", name, namespace)
			return true
		}
		if err != nil {
			k.Logger.Error(err)
			return false
		}
	}

	for key, value := range annotations {
		if key == "vaultingkube.io/managed" && value == "true" {
			return true
		}
	}

	return false
}

// SetCM will create or update a given config map
func (k *VKKube) SetCM(name string, namespace string, kv map[string]string) error {
	cm := &core_v1.ConfigMap{
		Data: kv,
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				"vaultingkube.io/managed": "true",
			},
		},
	}

	_, err := k.Client.CoreV1().ConfigMaps(namespace).Get(name, meta_v1.GetOptions{})
	if k8s_errors.IsNotFound(err) { // Check if we need to make new CM
		_, err := k.Client.CoreV1().ConfigMaps(namespace).Create(cm)
		if err != nil {
			return err
		}
	} else if err == nil { // No error getting CM, do an Update
		_, err := k.Client.CoreV1().ConfigMaps(namespace).Update(cm)
		if err != nil {
			return err
		}
	} else {
		return err
	}

	return nil
}

// SetSecret will create or update a given secret
func (k *VKKube) SetSecret(name string, namespace string, kv map[string]string) error {
	// Convert kv map[string]string to map[string][]byte
	// When creating/updating the secret it is base64 encoded for us already by kube
	byteKV := make(map[string][]byte)
	for key, value := range kv {
		byteKV[key] = []byte(value)
	}
	secret := &core_v1.Secret{
		Type: core_v1.SecretTypeOpaque,
		Data: byteKV,
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				"vaultingkube.io/managed": "true",
			},
		},
	}

	_, err := k.Client.CoreV1().Secrets(namespace).Get(name, meta_v1.GetOptions{})
	if k8s_errors.IsNotFound(err) { // If need to make new secret
		_, err := k.Client.CoreV1().Secrets(namespace).Create(secret)
		if err != nil {
			return err
		}
	} else if err == nil { // Secret found, do an Update
		_, err := k.Client.CoreV1().Secrets(namespace).Update(secret)
		if err != nil {
			return err
		}
	} else {
		return err
	}

	return nil
}

// DeleteOld deletes any CM's/secrets that are in kube but not Vault
func (k *VKKube) DeleteOld(mounts *VKVaultMounts) error {
	cms, err := k.Client.CoreV1().ConfigMaps("").List(meta_v1.ListOptions{})
	if err != nil {
		return err
	}
	secrets, err := k.Client.CoreV1().Secrets("").List(meta_v1.ListOptions{})
	if err != nil {
		return err
	}

	for _, cm := range cms.Items {
		if cm.Annotations["vaultingkube.io/managed"] != "true" {
			continue
		}
		found := false
		for _, mount := range *mounts {
			if mount.Secrets == nil {
				continue
			}
			for _, secret := range *mount.Secrets {
				if mount.Namespace == cm.Namespace &&
					secret.Name == cm.Name &&
					mount.SecretTypes == "configmaps" {
					// We found a cm
					found = true
				}
			}
		}
		if !found {
			err := k.DeleteCM(cm.Name, cm.Namespace)
			if err != nil {
				return err
			}
			k.Logger.Infof("Deleted old ConfigMap %s/%s", cm.Namespace, cm.Name)
		}
	}

	for _, secret := range secrets.Items {
		if secret.Annotations["vaultingkube.io/managed"] != "true" {
			continue
		}
		found := false
		for _, mount := range *mounts {
			if mount.Secrets == nil {
				continue
			}
			for _, mSecret := range *mount.Secrets {
				if mount.Namespace == secret.Namespace &&
					secret.Name == mSecret.Name &&
					mount.SecretTypes == "secrets" {
					// We found a secret
					found = true
				}
			}
		}
		if !found {
			err := k.DeleteSecret(secret.Name, secret.Namespace)
			if err != nil {
				return err
			}
			k.Logger.Infof("Deleted old Secret %s/%s", secret.Namespace, secret.Name)
		}
	}
	return nil
}

// DeleteCM delete's a given CM
func (k *VKKube) DeleteCM(name string, namespace string) error {
	return k.Client.CoreV1().ConfigMaps(namespace).Delete(name, &meta_v1.DeleteOptions{})
}

// DeleteSecret delete's a given Secret
func (k *VKKube) DeleteSecret(name string, namespace string) error {
	return k.Client.CoreV1().Secrets(namespace).Delete(name, &meta_v1.DeleteOptions{})
}
