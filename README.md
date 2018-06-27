# vaultingkube

![logo](resource/vaultingkube.png)

[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fsunshinekitty%2Fvaultingkube.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fsunshinekitty%2Fvaultingkube?ref=badge_shield)

Take config maps and secrets stored inside Hashicorp Vault and sync them to your Kubernetes cluster.

## How it works

After Vaultingkube is running in your cluster it will look at the Vault server configured via the Vault client config options.  Based on the `VK_VAULT_ROOT_MOUNT_PATH` Vaultingkube will read all kv secrets it has access to and reference any with a matching mount path.  EG if `VK_VAULT_ROOT_MOUNT_PATH` is set to `vaultingkube/my-cluster` it will look at all mounts at `vaultingkube/my-cluster/*`.

The type of secret and namespace are configured in the mount path as well.  It looks like `[VK_VAULT_ROOT_MOUNT_PATH]/[NAMESPACE]/[SECRET_TYPE]/[NAME]`.  If I wanted to create a configmap in the `default` namespace named `tom` I would create a kv secret at `[VK_VAULT_ROOT_MOUNT_PATH]/default/configmaps/tom`.

Vaultingkube will only overwrite, manage, or delete ConfigMaps and Secrets that have the annotation `vaultingkube.io/managed: "true"` set.  You will need to manually set this on existing Secrets and Configmaps for Vaultingkube to take over.

Vaultingkube does not have any logic to determine if Vault has changed, and so it uses the `VK_SYNC_PERIOD` environment variable to determine how frequently (in seconds) it sends update requests to Kubernetes to stay in sync with Vault.

By default Vaultingkube will delete ConfigMaps and Secrets that exist in Kubernetes and **not** Vault that have the annotation of `vaultingkube.io/managed: "true"`.  To turn this off set the environment variable `VK_DELETE_OLD` to `"false"`.

## Demo

<img src="resource/demo.gif" width="600px" />

## Deployment

### Manual

On lines 25-34 in [deployment/003-deployment.yaml](deployment/003-deployment.yaml) are configurable environment variables that will need to be set on a per deployment basis.  After these are updated for your environment run `kubectl apply -f deployment/`.  See [environment variables](#environment-variables).

**If you are not using RBAC** then just run 

```
kubectl apply -f deployment/001-common.yaml
kubectl apply -f deployment/003-deployment.yaml
```

### Helm

```
helm repo add incubator https://kubernetes-charts-incubator.storage.googleapis.com/
helm install incubator/vaultingkube
```

## Environment Variables

| Variable                 | Explanation                                                         |
|--------------------------|---------------------------------------------------------------------|
| VK_DELETE_OLD            | Delete CM/Secret's that are in Kube but not Vault [Default: true]   |
| VK_SYNC_PERIOD           | How many seconds to wait in between syncs [Default: 300]            |
| VK_VAULT_ROOT_MOUNT_PATH | Path to configmaps and secrets [Example: vaultingkube/cluster-name] |

## Client Versions

You should be able to infer the supported versions of Kubernetes and Vault that will work with this program from these.

- client-go: 5.0.1
- vault: 0.9.0

## Vault permissions

**Note** replace VK_VAULT_ROOT_MOUNT_PATH with the real value
```
{
  "path": {
      "sys/mounts": {
          "capabilities": [
            "read"
          ]
        }
    }
 }

{
  "path": {
      "${VK_VAULT_ROOT_MOUNT_PATH}/*": {
          "capabilities": [
            "read",
            "list"
          ]
        }
    }
}
```

## License

[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fsunshinekitty%2Fvaultingkube.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fsunshinekitty%2Fvaultingkube?ref=badge_large)
