resource "hcloud_network" "main" {
  name     = "main"
  ip_range = "10.0.0.0/16"
}

resource "hcloud_network_subnet" "main" {
  network_id   = hcloud_network.main.id
  type         = "cloud"
  network_zone = "eu-central"
  ip_range     = "10.0.1.0/24"
}

resource "hcloud_server" "example" {
  name        = "example"
  image       = "ubuntu-24.04"
  server_type = "cx22"
  location    = "nbg1"

  network {
    network_id = hcloud_network.main.id
  }

  depends_on = [hcloud_network_subnet.main]
}
