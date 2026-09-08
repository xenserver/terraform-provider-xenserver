provider "xenserver" {
  host         = "https://192.0.2.1"
  username     = "root"
  password     = var.password
  insecure     = false
  ca_cert_path = "/opt/cert.pem"
}
