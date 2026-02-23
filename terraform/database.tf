# Create a Cloud SQL for PostgreSQL instance
resource "google_sql_database_instance" "db_instance" {
  name             = "ozmadeback-db"
  database_version = "POSTGRES_14"
  region           = var.region

  settings {
    tier = "db-g1-small"
    ip_configuration {
      ipv4_enabled = true

      authorized_networks {
        name  = "vm-app-server"
        value = google_compute_address.static_ip.address
      }
    }
  }
}

output "db_ip" {
  value = google_sql_database_instance.db_instance.public_ip_address
}

# Create the database within the instance
resource "google_sql_database" "database" {
  name     = "ozmadeback_db"
  instance = google_sql_database_instance.db_instance.name
}

# Create a user for the application
resource "google_sql_user" "db_user" {
  name     = "ozmade_user"
  instance = google_sql_database_instance.db_instance.name
  password = var.db_password
}