resource "tailscale_tailnet_key" "ci" {
  ephemeral     = true
  preauthorized = true
  expiry        = 3600
  description   = "single-use CI key, reusable defaults off"
}
