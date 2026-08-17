# access defaults to public on vercel_blob_store, so a store that never sets it serves every
# object it holds to anyone holding the URL.
resource "vercel_blob_store" "uploads" {
  name = "customer-uploads"
}
