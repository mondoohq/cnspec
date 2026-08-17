resource "vercel_blob_store" "uploads" {
  name   = "customer-uploads"
  access = "public"
}
