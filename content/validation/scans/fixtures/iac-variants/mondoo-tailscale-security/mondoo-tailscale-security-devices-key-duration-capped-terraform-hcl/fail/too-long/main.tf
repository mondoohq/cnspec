resource "tailscale_tailnet_settings" "this" {
  devices_approval_on       = true
  devices_key_duration_days = 365
}
