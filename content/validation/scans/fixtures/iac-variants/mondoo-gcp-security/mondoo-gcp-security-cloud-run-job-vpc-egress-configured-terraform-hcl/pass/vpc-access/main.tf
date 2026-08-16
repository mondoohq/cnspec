# Compliant: the job task template has a vpc_access block for egress control.
resource "google_cloud_run_v2_job" "pass_example" {
  name     = "batch-job"
  location = "us-central1"

  template {
    template {
      containers {
        image = "us-docker.pkg.dev/my-project/repo/batch:latest"
      }

      vpc_access {
        connector = "projects/my-project/locations/us-central1/connectors/my-connector"
        egress    = "ALL_TRAFFIC"
      }
    }
  }
}
