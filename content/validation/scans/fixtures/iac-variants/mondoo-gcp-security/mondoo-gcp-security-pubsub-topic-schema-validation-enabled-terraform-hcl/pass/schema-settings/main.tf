# Compliant: topic has a schema_settings block enforcing message validation.
resource "google_pubsub_schema" "example" {
  name       = "my-schema"
  type       = "AVRO"
  definition = "{\"type\":\"record\",\"name\":\"Event\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}"
}

resource "google_pubsub_topic" "pass_example" {
  name = "my-topic"

  schema_settings {
    schema   = google_pubsub_schema.example.id
    encoding = "JSON"
  }
}
