resource "tailscale_tailnet_key" "ci" {
  reusable      = false
  ephemeral     = true
  preauthorized = true
  expiry        = 7776000
  description   = "ephemeral CI runner key"
}
