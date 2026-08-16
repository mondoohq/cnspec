resource "tailscale_device_authorization" "example" {
  device_id  = data.tailscale_device.example.id
  authorized = true
}

data "tailscale_device" "example" {
  name = "laptop.example.ts.net"
}
