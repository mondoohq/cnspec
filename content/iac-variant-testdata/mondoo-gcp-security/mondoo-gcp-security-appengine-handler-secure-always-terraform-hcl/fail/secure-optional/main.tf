# Non-compliant: handler allows insecure access (SECURE_OPTIONAL).
resource "google_app_engine_standard_app_version" "fail_example" {
  version_id = "v1"
  service    = "default"
  runtime    = "nodejs20"

  entrypoint {
    shell = "node ./app.js"
  }

  handlers {
    url_regex      = "/.*"
    security_level = "SECURE_OPTIONAL"

    script {
      script_path = "auto"
    }
  }
}
