# A line with no entry in the lifecycle table fails rather than passing unexamined.
resource "vercel_project" "storefront" {
  name         = "storefront"
  node_version = "99.x"
}
