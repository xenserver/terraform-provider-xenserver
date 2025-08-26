provider "xenserver" {
  host             = "https://192.0.2.1"
  username         = "root"
  password         = var.password
  skip_verify      = false
  server_cert_path = "/opt/cert.pem"
}
