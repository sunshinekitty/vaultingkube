package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/sunshinekitty/vaultingkube/comms"
)

func main() {
	rootMountPath := os.Getenv("VK_VAULT_ROOT_MOUNT_PATH")
	if rootMountPath == "" {
		log.Fatal("Must set VK_VAULT_ROOT_MOUNT_PATH")
	}

	syncPeriod := os.Getenv("VK_SYNC_PERIOD")
	if syncPeriod == "" {
		syncPeriod = "300"
	}

	syncPeriodInt, err := strconv.Atoi(syncPeriod)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Started with sync period every %s seconds\n", syncPeriod)
	for range time.Tick(time.Second * time.Duration(syncPeriodInt)) {
		run()
	}
}

func run() {
	logger := logrus.New()

	vkVault, err := comms.NewVKVaultClient()
	if err != nil {
		logger.Fatal(err)
	}

	vkKube, err := comms.NewVKKubeClient(logger)
	if err != nil {
		logger.Fatal(err)
	}

	mounts, err := vkVault.GetMounts(os.Getenv("VK_VAULT_ROOT_MOUNT_PATH"))
	if err != nil {
		logger.Fatal(err)
	} else {
		for _, mount := range *mounts {
			if mount.Secrets != nil {
				for _, secret := range *mount.Secrets {
					if vkKube.IsManaged(secret.Name, mount.SecretTypes, mount.Namespace) {
						if mount.SecretTypes == "secrets" {
							err := vkKube.SetSecret(secret.Name, mount.Namespace, secret.Pairs)
							if err != nil {
								logger.Error(err)
							} else {
								logger.Infof("Set Secret for %s/%s", mount.Namespace, secret.Name)
							}
						} else if mount.SecretTypes == "configmaps" {
							err := vkKube.SetCM(secret.Name, mount.Namespace, secret.Pairs)
							if err != nil {
								logger.Error(err)
							} else {
								logger.Infof("Set ConfigMap for %s/%s", mount.Namespace, secret.Name)
							}
						}
					} else {
						if mount.SecretTypes == "secrets" {
							logger.Infof("Secret %s in namespace %s is not managed by VaultingKube, ignoring", secret.Name, mount.Namespace)
						} else if mount.SecretTypes == "configmaps" {
							logger.Infof("ConfigMap %s in namespace %s is not managed by VaultingKube, ignoring", secret.Name, mount.Namespace)
						}
					}
				}
			}
		}

		deleteOld := os.Getenv("VK_DELETE_OLD")
		if deleteOld == "" || deleteOld == "true" {
			err := vkKube.DeleteOld(mounts)
			if err != nil {
				logger.Fatal(err)
			}
		}
	}
}
