package comms

import (
	"errors"
	"fmt"
	"strings"

	vault "github.com/hashicorp/vault/api"
)

var (
	ErrInvalidVaultMount = errors.New("vault mount is invalid")
	ErrInvalidSecretType = errors.New("vault secret is invalid type")
)

// VKVault is a wrapper around Vault
type VKVault struct {
	Client *vault.Client
}

// VKVaultMounts is a slice of VKVaultMount's
type VKVaultMounts []VKVaultMount

// VKVaultMount is a representation of a mount we care about
type VKVaultMount struct {
	MountPath    string
	MountPointer *vault.MountOutput
	Namespace    string
	SecretTypes  string
	Secrets      *VKVaultSecrets
}

// VKVaultSecrets is a slice of VKVaultSecret's
type VKVaultSecrets []VKVaultSecret

// VKVaultSecret is a kv secret stored in vault
type VKVaultSecret struct {
	Name  string
	Pairs map[string]string
}

// NewVKVaultClient returns a new VKVault client
func NewVKVaultClient() (*VKVault, error) {
	config := vault.DefaultConfig()
	client, err := vault.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &VKVault{
		Client: client,
	}, nil
}

// GetMounts will return a list of VKVaultMounts based on mountPath
func (v *VKVault) GetMounts(mountPath string) (*VKVaultMounts, error) {
	mountPath = strings.Trim(mountPath, "/")
	mounts := new(VKVaultMounts)
	mountMap, err := v.Client.Sys().ListMounts()
	if err != nil {
		return nil, err
	}

	for mount, pointer := range mountMap {
		if strings.HasPrefix(mount, mountPath) && pointer.Type == "kv" {
			subMount := strings.Split(mount, mountPath)[1]
			subMount = strings.Trim(subMount, "/")
			subMountSlice := strings.Split(subMount, "/")
			if len(subMountSlice) != 2 {
				ErrInvalidVaultMount = fmt.Errorf("Mount %s is invalid", mount)
				return nil, ErrInvalidVaultMount
			}
			namespace := subMountSlice[0]
			secretTypes := subMountSlice[1]
			if secretTypes != "configmaps" && secretTypes != "secrets" {
				ErrInvalidSecretType = fmt.Errorf("Secret %s is invalid", secretTypes)
				return nil, ErrInvalidSecretType
			}
			vaultMount := &VKVaultMount{
				MountPath:    mount,
				MountPointer: pointer,
				Namespace:    namespace,
				SecretTypes:  secretTypes,
			}
			vaultMount.populateSecrets(v)
			*mounts = append(*mounts, *vaultMount)
		}
	}

	return mounts, nil
}

// populateSecrets will return VKVaultSecrets when given a VKVaultMount
func (m *VKVaultMount) populateSecrets(v *VKVault) (*VKVaultMount, error) {
	returnSecrets := new(VKVaultSecrets)
	secrets, err := v.Client.Logical().List(m.MountPath)
	if err != nil {
		return nil, err
	}
	if secrets == nil {
		return nil, nil
	}
	for _, data := range secrets.Data["keys"].([]interface{}) {
		secretName := data.(string)
		secretMap, err := v.Client.Logical().Read(m.MountPath + "/" + secretName)
		if err != nil {
			return nil, err
		}
		appendSecret := &VKVaultSecret{
			Name:  secretName,
			Pairs: make(map[string]string),
		}
		if secretMap != nil {
			for key, value := range secretMap.Data {
				appendSecret.Pairs[key] = value.(string)
			}
		}
		*returnSecrets = append(*returnSecrets, *appendSecret)
	}
	m.Secrets = returnSecrets
	return m, nil
}
