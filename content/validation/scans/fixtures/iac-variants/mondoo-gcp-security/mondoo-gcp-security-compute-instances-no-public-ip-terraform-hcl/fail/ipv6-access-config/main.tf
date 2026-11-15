# Non-compliant: ipv6_access_config assigns an external IPv6 address.
resource "google_compute_instance" "dual_stack" {
  name         = "dual-stack-vm"
  machine_type = "e2-medium"
  zone         = "us-central1-a"

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-12"
    }
  }

  network_interface {
    subnetwork = google_compute_subnetwork.dual_stack.id
    stack_type = "IPV4_IPV6"

    ipv6_access_config {
      network_tier = "PREMIUM"
    }
  }
}
