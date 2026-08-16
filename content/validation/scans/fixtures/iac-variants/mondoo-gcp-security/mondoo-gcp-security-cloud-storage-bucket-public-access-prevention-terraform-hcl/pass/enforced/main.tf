# Compliant: public access prevention is enforced.
resource "google_storage_bucket" "secure" {
  name                     = "my-secure-bucket"
  location                 = "US"
  public_access_prevention = "enforced"
}
