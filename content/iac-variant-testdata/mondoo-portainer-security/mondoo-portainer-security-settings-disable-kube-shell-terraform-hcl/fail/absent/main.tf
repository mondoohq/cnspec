# disable_kube_shell is not set, so the Kubernetes web shell stays enabled.
resource "portainer_settings" "this" {
  snapshot_interval = "5m"
}
