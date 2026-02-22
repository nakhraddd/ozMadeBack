# Create a GCS bucket for storing images and other assets
resource "google_storage_bucket" "storage_bucket" {
  name          = "${var.project_id}-ozmadeback-assets"
  location      = var.region
  force_destroy = true # Allows bucket deletion even if it has objects

  uniform_bucket_level_access = true

  # This setting controls public access prevention. "inherited" is the default.
  public_access_prevention = "inherited"
}

# Grant allUsers the Storage Object Viewer role to make objects public
resource "google_storage_bucket_iam_member" "public_rule" {
  bucket = google_storage_bucket.storage_bucket.name
  role   = "roles/storage.objectViewer"
  member = "allUsers"
}
