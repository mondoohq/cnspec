resource "cloudflare_api_token" "example" {
  name = "ci-deploy-token"

  policies = [{
    effect = "allow"
    permission_groups = [{
      id = "1a71c399035b4950a1bd1466b1e4f420"
    }]
    resources = {
      "com.cloudflare.api.account.0123456789abcdef0123456789abcdef" = "*"
    }
  }]

  condition = {
    request_ip = {
      in = ["198.51.100.0/24", "203.0.113.10/32"]
    }
  }
}
