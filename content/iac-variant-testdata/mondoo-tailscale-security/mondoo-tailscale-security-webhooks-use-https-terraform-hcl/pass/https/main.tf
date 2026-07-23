resource "tailscale_webhook" "example" {
  endpoint_url  = "https://example.com/webhooks/tailscale"
  provider_type = "slack"
  subscriptions = ["nodeCreated", "userNeedsApproval"]
}
