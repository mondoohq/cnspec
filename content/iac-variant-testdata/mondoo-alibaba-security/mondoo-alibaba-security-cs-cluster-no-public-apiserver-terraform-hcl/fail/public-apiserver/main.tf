# A public SLB in front of the API server exposes the cluster control plane to
# the internet.
resource "alicloud_cs_managed_kubernetes" "prod" {
  name                 = "prod-ack"
  worker_vswitch_ids   = [alicloud_vswitch.k8s.id]
  pod_cidr             = "172.20.0.0/16"
  service_cidr         = "172.21.0.0/20"
  slb_internet_enabled = true
}
