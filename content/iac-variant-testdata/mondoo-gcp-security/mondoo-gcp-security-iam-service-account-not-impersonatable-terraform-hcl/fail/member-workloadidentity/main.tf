# Non-compliant: iam_member grants Workload Identity User, allowing impersonation.
resource "google_service_account_iam_member" "fail_example" {
  service_account_id = "projects/my-project/serviceAccounts/my-sa@my-project.iam.gserviceaccount.com"
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:my-project.svc.id.goog[ns/ksa]"
}
