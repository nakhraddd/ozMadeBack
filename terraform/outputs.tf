output "instance_ip" {
  description = "The static IP address of the VM instance."
  value       = google_compute_address.static_ip.address
}

output "db_connection_name" {
  description = "The connection name of the Cloud SQL instance for use by the application."
  value       = google_sql_database_instance.db_instance.connection_name
}

output "gcs_bucket_name" {
  description = "The name of the GCS bucket for application assets."
  value       = google_storage_bucket.storage_bucket.name
}