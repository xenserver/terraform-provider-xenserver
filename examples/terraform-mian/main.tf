terraform {
  required_providers {
    xenserver = {
      source = "xenserver/xenserver"
    }
  }
}

provider "xenserver" {
  host     = "https://10.70.40.100"
  username = "root"
  password = "BOfpcNyZ5cMe"
}
