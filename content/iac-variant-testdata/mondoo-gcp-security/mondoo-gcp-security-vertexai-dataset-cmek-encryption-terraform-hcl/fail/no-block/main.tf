resource "google_vertex_ai_dataset" "fail" {
  display_name        = "terraform-dataset"
  metadata_schema_uri = "gs://google-cloud-aiplatform/schema/dataset/metadata/image_1.0.0.yaml"
  region              = "us-central1"
}
