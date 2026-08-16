# Compliant: disk encrypted with a customer-supplied RSA-wrapped key.
resource "google_compute_disk" "example" {
  name = "csek-rsa-disk"
  type = "pd-balanced"
  zone = "us-central1-a"
  size = 100

  disk_encryption_key {
    rsa_encrypted_key = "ieCx/NcW06PcT7Ep1X6LUTc/hLvUDYyzSZPPVCVPTVEohpeHASqC8uw5TzyO9U+Fka9JFHz0mBibXUInrC/jEk014kCK/NPjYgEMOyssZ4ZINPKxlUh2zn1bV+MCaTICrdmuSBTWlUUiFoDD6PYznLwh8ZNdaheCeZ8ewEXgFQ8V+sDroLaN3Xs3MDTXQEMMoNUXMCZEIpg9Vtp9x2oeQ5lAbtt7bYAAHf5l+gJWw3sUfs0/Glw5fpdjT8Uggrr+RMZezGrltJEF293rvTIjWOEB3z5OHyHwZfXY="
    sha256            = "2v5x3JQzHZgL8b8Q5uHc+3Kpl0m9WvJm0kFQxQ5r3Y="
  }
}
