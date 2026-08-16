resource "cloudflare_account" "this" {
  name = "Example Account"
  settings = {
    enforce_twofactor = true
  }
}
