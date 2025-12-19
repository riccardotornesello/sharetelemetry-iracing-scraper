// PROJECT

variable "region" {
  description = "The Google Cloud region where the function will be deployed."
  type        = string
  default     = "europe-west1"
}

variable "project_id" {
  description = "The Google Cloud project ID."
  type        = string
}


// IRACING

variable "iracing_client_id" {
  description = "The client ID used to authenticate with the iRacing API."
  type        = string
}

variable "iracing_client_secret" {
  description = "The client secret used to authenticate with the iRacing API."
  type        = string
}

variable "iracing_username" {
  description = "The email address used to authenticate with the iRacing API."
  type        = string
}

variable "iracing_password" {
  description = "The password used to authenticate with the iRacing API."
  type        = string
}
