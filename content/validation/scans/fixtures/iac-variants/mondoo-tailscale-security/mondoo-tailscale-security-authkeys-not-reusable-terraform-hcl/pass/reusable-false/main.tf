resource "tailscale_tailnet_key" "ci" {
  reusable      = false
  ephemeral     = true
  preauthorized = true
  expiry        = 3600
  description   = "single-use CI key"
}
