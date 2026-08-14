# The dedicated (non-managed) cluster type is covered by the same control.
resource "alicloud_cs_kubernetes" "legacy" {
  name                 = "legacy-ack"
  master_vswitch_ids   = [alicloud_vswitch.k8s.id, alicloud_vswitch.k8s_b.id, alicloud_vswitch.k8s_c.id]
  worker_vswitch_ids   = [alicloud_vswitch.k8s.id]
  master_instance_types = ["ecs.n4.large", "ecs.n4.large", "ecs.n4.large"]
  slb_internet_enabled = true
}
