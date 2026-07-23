resource "tailscale_tailnet_key" "ci" {
  reusable      = false
  ephemeral     = true
  preauthorized = true
  expiry        = 0
  description   = "never-expiring CI runner key"
}
