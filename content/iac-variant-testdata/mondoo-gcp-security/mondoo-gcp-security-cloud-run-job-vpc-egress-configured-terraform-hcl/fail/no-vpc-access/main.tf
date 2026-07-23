# Non-compliant: the job task template has no vpc_access block, so egress is
# not routed through a VPC connector.
resource "google_cloud_run_v2_job" "fail_example" {
  name     = "batch-job"
  location = "us-central1"

  template {
    template {
      containers {
        image = "us-docker.pkg.dev/my-project/repo/batch:latest"
      }
    }
  }
}
