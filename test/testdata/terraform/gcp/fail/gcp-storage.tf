# The following examples will fail the google-storage-no-public-access check.

resource "google_storage_bucket_iam_binding" "allAuthenticatedUsers" {
	bucket = google_storage_bucket.default.name
	role = "roles/storage.admin"
	members = [
		"allAuthenticatedUsers",
	]
}

resource "google_storage_bucket_iam_binding" "allUsers" {
	bucket = google_storage_bucket.default.name
	role = "roles/storage.admin"
	members = [
		"allUsers",
	]
}

# The following example will fail the google-storage-enable-ubla check.

resource "google_storage_bucket" "static-site" {
	name          = "image-store.com"
	location      = "EU"
	force_destroy = true
	
	uniform_bucket_level_access = false
	
	website {
		main_page_suffix = "index.html"
		not_found_page   = "404.html"
	}
	cors {
		origin          = ["http://image-store.com"]
		method          = ["GET", "HEAD", "PUT", "POST", "DELETE"]
		response_header = ["*"]
		max_age_seconds = 3600
	}
}