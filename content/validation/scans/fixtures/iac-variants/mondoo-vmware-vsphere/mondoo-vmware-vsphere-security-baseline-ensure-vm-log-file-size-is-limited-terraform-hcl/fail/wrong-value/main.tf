resource "vsphere_virtual_machine" "web" {
  name             = "web-01"
  resource_pool_id = data.vsphere_compute_cluster.cluster.resource_pool_id
  datastore_id     = data.vsphere_datastore.ds.id
  num_cpus         = 2
  memory           = 4096
  guest_id         = "ubuntu64Guest"
  firmware         = "efi"

  network_interface {
    network_id = data.vsphere_network.net.id
  }

  disk {
    label = "disk0"
    size  = 40
  }

  extra_config = {
    "log.rotateSize" = "2048000"
  }
}
