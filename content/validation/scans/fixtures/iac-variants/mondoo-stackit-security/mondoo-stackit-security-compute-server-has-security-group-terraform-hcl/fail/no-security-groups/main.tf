resource "stackit_network_interface" "app" {
  project_id = "8e1e0b09-2f5a-4c0d-9f04-1b6b2a3c4d5e"
  network_id = "5f6a1c22-7d3b-4f18-9a2c-0b7e8d9f1a2b"
}

resource "stackit_server" "app" {
  project_id   = "8e1e0b09-2f5a-4c0d-9f04-1b6b2a3c4d5e"
  name         = "app-01"
  machine_type = "g2i.1"
  boot_volume = {
    size        = 64
    source_type = "image"
    source_id   = "3a1c4e7b-92d5-4f6a-b8c1-2d3e4f5a6b7c"
  }
  network_interfaces = [stackit_network_interface.app.network_interface_id]
}
