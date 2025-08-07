provider "xenserver" {
  host     = "https://192.0.2.1"
  username = "root"
  password = var.password
  server_cert_path = "/opt/cert.pem"
}
