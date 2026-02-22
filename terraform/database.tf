# Create a Cloud SQL for PostgreSQL instance
resource "google_sql_database_instance" "db_instance" {
  name             = "ozmadeback-db-instance"
  database_version = "POSTGRES_14"
  region           = var.region

  settings {
    tier = "db-g1-small" # A small, cost-effective tier for development
    ip_configuration {
      ipv4_enabled = true
      # Allow public access for simplicity, but restrict with authorized_networks
      # For production, prefer using a private IP and a VPC connector.
    }
  }
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