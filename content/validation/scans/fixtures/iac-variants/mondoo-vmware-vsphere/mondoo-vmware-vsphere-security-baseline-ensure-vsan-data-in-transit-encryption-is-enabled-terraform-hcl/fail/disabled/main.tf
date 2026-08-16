resource "vsphere_compute_cluster" "cluster" {
  name          = "prod-cluster"
  datacenter_id = data.vsphere_datacenter.dc.id

  host_system_ids = [data.vsphere_host.host.id]

  drs_enabled          = true
  drs_automation_level = "fullyAutomated"

  ha_enabled = true

  vsan_enabled = true
  vsan_dit_encryption_enabled = false
}
