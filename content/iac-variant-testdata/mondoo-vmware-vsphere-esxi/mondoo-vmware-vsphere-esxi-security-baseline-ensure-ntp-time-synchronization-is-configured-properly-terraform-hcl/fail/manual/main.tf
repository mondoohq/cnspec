resource "vsphere_host" "esxi01" {
  hostname = "esxi-01.example.com"
  username = "root"
  password = var.esxi_password

  datacenter = data.vsphere_datacenter.dc.id
  lockdown   = "normal"

  services {
    ntpd {
      enabled     = true
      policy      = "off"
      ntp_servers = ["0.pool.ntp.org"]
    }
  }
}
