# Non-compliant: public access prevention is inherited, not enforced.
resource "google_storage_bucket" "loose" {
  name                     = "my-loose-bucket"
  location                 = "US"
  public_access_prevention = "inherited"
}
