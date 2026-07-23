resource "vsphere_host" "esxi01" {
  hostname = "esxi-01.example.com"
  username = "root"
  password = var.esxi_password

  datacenter = data.vsphere_datacenter.dc.id
}
