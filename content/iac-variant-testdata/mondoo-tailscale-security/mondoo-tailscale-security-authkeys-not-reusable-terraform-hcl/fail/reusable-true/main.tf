resource "tailscale_tailnet_key" "shared" {
  reusable      = true
  ephemeral     = false
  preauthorized = true
  expiry        = 7776000
  description   = "shared reusable enrollment key"
}
