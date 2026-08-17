# expires_at is optional, so omitting it mints a token that stays valid until someone revokes it.
resource "vercel_user_token" "deploy" {
  name = "ci-deploy"
}
