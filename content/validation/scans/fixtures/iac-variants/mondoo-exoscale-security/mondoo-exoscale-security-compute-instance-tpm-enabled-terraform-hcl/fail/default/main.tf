data "exoscale_template" "ubuntu" {
  zone = "ch-gva-2"
  name = "Linux Ubuntu 24.04 LTS 64-bit"
}

resource "exoscale_compute_instance" "web" {
  zone        = "ch-gva-2"
  name        = "web-1"
  template_id = data.exoscale_template.ubuntu.id
  type        = "standard.small"
  disk_size   = 10
}
