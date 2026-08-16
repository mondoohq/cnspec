resource "google_project_iam_custom_role" "inventory_reader" {
  project     = var.project_id
  role_id     = "inventoryReader"
  title       = "Inventory Reader"
  description = "Read-only access used by the asset inventory job"
  stage       = "GA"

  permissions = [
    "compute.instances.list",
    "compute.disks.list",
    "storage.objects.get",
    "storage.buckets.list",
  ]
}
