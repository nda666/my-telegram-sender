

data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "ariga.io/atlas-provider-gorm",
    "load",
    "--path", "./internal/models",
    "--dialect", "mysql", // | postgres | sqlite | sqlserver
  ]
}
variable "envfile" {
    type    = string
    default = ".env"
}

locals {
    envfile = {
        for line in split("\n", file(var.envfile)): split("=", line)[0] => regex("=(.*)", line)[0]
        if !startswith(line, "#") && length(split("=", line)) > 1
    }
}
env "gorm" {
  src = data.external_schema.gorm.url
  url = "mysql://gotele:gotele@localhost:3306/gotele_prod"
  dev = "mysql://gotele:gotele@localhost:3306/gotele_dev"
  migration {
    dir = "file://migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}

env "prod" {
  src = data.external_schema.gorm.url
  url = "mysql://gotele:gotele@localhost:3306/gotele_prod"
  dev = "mysql://gotele:gotele@localhost:3306/gotele_dev"
  migration {
    dir = "file://migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}