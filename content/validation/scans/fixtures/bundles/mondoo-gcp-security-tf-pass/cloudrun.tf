# Cloud Run fixture for the tf-pass test bundle.
#
# The service uses CMEK, a custom service account, restricted ingress, and VPC
# access through a Serverless VPC Access connector.

resource "google_vpc_access_connector" "run_connector" {
  name          = "run-conn-${random_id.rnd.hex}"
  region        = var.region
  ip_cidr_range = "10.9.0.0/28"
  network       = google_compute_network.vpc_network.name
}

resource "google_service_account" "cloud_run_sa" {
  account_id   = "cloud-run-sa-${random_id.rnd.hex}"
  display_name = "Cloud Run Service Account"
}

resource "google_cloud_run_v2_service" "api" {
  name     = "api-${random_id.rnd.hex}"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"

  template {
    encryption_key  = google_kms_crypto_key.key.id
    service_account = google_service_account.cloud_run_sa.email

    vpc_access {
      connector = google_vpc_access_connector.run_connector.id
      egress    = "ALL_TRAFFIC"
    }

    containers {
      image = "us-docker.pkg.dev/cloudrun/container/hello"
    }
  }
}
