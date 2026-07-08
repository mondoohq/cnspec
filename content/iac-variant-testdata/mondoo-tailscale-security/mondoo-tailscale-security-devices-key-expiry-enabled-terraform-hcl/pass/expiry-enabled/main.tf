resource "tailscale_device_key" "example" {
  device_id           = data.tailscale_device.example.id
  key_expiry_disabled = false
}

data "tailscale_device" "example" {
  name = "server.example.ts.net"
}
