resource "tailscale_webhook" "example" {
  endpoint_url  = "http://example.com/webhooks/tailscale"
  provider_type = "slack"
  subscriptions = ["nodeCreated", "userNeedsApproval"]
}
