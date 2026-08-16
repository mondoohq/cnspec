resource "tailscale_tailnet_settings" "this" {
  devices_auto_updates_on   = true
  devices_key_duration_days = 90
}
