# The following example will pass the google-iam-no-folder-level-default-service-account-assignment check.
resource "google_service_account" "test" {
	account_id   = "account123"
	display_name = "account123"
}
			  
resource "google_folder_iam_member" "folder-123" {
	folder = "folder-123"
	role    = "roles/whatever"
	member  = "serviceAccount:${google_service_account.test.email}"
}

resource "google_folder_iam_binding" "folder-123" {
	folder = "folder-123"
	role    = "roles/custom-role"
}

# The following example will pass the google-iam-no-privileged-service-accounts check.
resource "google_service_account" "test" {
	account_id   = "account123"
	display_name = "account123"
}

resource "google_project_iam_member" "project" {
	project = "your-project-id"
	role    = "roles/logging.logWriter"
	member  = "serviceAccount:${google_service_account.test.email}"
}