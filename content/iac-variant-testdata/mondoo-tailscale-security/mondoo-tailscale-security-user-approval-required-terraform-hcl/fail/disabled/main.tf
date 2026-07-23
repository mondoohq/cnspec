resource "tailscale_tailnet_settings" "this" {
  users_approval_on   = false
  devices_approval_on = true
}
