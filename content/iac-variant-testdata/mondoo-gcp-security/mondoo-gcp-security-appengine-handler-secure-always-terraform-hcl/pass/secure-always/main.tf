# Compliant: every handler enforces SECURE_ALWAYS.
resource "google_app_engine_standard_app_version" "pass_example" {
  version_id = "v1"
  service    = "default"
  runtime    = "nodejs20"

  entrypoint {
    shell = "node ./app.js"
  }

  handlers {
    url_regex      = "/.*"
    security_level = "SECURE_ALWAYS"

    script {
      script_path = "auto"
    }
  }
}
