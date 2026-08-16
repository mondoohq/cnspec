data "vsphere_datacenter" "dc" {
  name = "dc1"
}

data "vsphere_host" "host" {
  name          = "esxi-01.example.com"
  datacenter_id = data.vsphere_datacenter.dc.id
}

resource "vsphere_host_virtual_switch" "switch" {
  name             = "vSwitchTerraformTest"
  host_system_id   = data.vsphere_host.host.id
  network_adapters = ["vmnic0", "vmnic1"]
  active_nics      = ["vmnic0"]
  standby_nics     = ["vmnic1"]
}

resource "vsphere_host_port_group" "pg" {
  name                = "PGTerraformTest"
  host_system_id      = data.vsphere_host.host.id
  virtual_switch_name = vsphere_host_virtual_switch.switch.name
  vlan_id             = 1
}
