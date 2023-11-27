# The following example will fail the google-iam-no-folder-level-default-service-account-assignment check.

resource "google_folder_iam_member" "folder-123" {
	folder = "folder-123"
	role    = "roles/my-role"
	member  = "123-compute@developer.gserviceaccount.com"
}

resource "google_folder_iam_member" "folder-456" {
	folder = "folder-456"
	role    = "roles/my-role"
	member  = "123@appspot.gserviceaccount.com"
}

data "google_compute_default_service_account" "default" {
}

resource "google_folder_iam_member" "folder-789" {
	folder = "folder-789"
	role    = "roles/my-role"
	member  = data.google_compute_default_service_account.default.id
}

# The following example will fail the google-iam-no-folder-level-service-account-impersonation check

resource "google_folder_iam_binding" "folder-123" {
	folder = "folder-123"
	role    = "roles/iam.serviceAccountUser"
}

# The following example will fail the google-iam-no-privileged-service-accounts check.

resource "google_service_account" "test" {
  account_id   = "account123"
  display_name = "account123"
}

resource "google_project_iam_member" "project" {
	project = "your-project-id"
	role    = "roles/owner"
	member  = "serviceAccount:${google_service_account.test.email}"
}