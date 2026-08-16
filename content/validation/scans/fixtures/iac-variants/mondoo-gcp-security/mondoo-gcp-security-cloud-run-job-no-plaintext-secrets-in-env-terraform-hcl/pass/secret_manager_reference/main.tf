resource "google_cloud_run_v2_job" "default" {
  name     = "example-job"
  location = "us-central1"

  template {
    template {
      containers {
        image = "us-docker.pkg.dev/cloudrun/container/hello"

        env {
          name  = "LOG_LEVEL"
          value = "info"
        }

        env {
          name = "DB_PASSWORD"
          value_source {
            secret_key_ref {
              secret  = "db-password"
              version = "latest"
            }
          }
        }
      }
    }
  }
}
