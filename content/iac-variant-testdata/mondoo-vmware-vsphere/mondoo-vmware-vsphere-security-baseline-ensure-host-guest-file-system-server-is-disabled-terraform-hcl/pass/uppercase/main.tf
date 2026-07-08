# Compliant: isolation.tools.hgfsServerSet.disable = "TRUE".
resource "vsphere_virtual_machine" "web" {
  name             = "web-01"
  resource_pool_id = data.vsphere_compute_cluster.cluster.resource_pool_id
  datastore_id     = data.vsphere_datastore.datastore.id
  num_cpus         = 2
  memory           = 4096
  guest_id         = "ubuntu64Guest"

  network_interface {
    network_id = data.vsphere_network.network.id
  }

  disk {
    label = "disk0"
    size  = 40
  }

  extra_config = {
    "isolation.tools.hgfsServerSet.disable" = "TRUE"
  }
}
