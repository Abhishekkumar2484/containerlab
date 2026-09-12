A production-style DevOps and Cloud deployment project built around a containerized Go REST API, PostgreSQL, Docker, GitHub Actions, Trivy, GHCR, AWS EC2, Nginx, Docker Compose, and Terraform.

Overview

ContainerLab demonstrates an end-to-end DevOps workflow:

Developer
   │
   │ git push
   ▼
GitHub
   │
   ▼
GitHub Actions
   ├── Go dependency download
   ├── Go tests
   ├── Go build
   ├── Docker image build
   ├── Trivy security scan
   └── Push image to GHCR
            │
            ▼
   GitHub Container Registry
            │
            ▼
        AWS EC2
            │
        Docker Compose
            │
      ┌─────┼─────┐
      ▼     ▼     ▼
    Nginx   API  PostgreSQL
     :80   :8080    :5432
      │
      ▼
    Public HTTP endpoint

Current Architecture

The application is deployed on an AWS EC2 instance in ap-south-1 (Mumbai).

                    INTERNET
                       │
                    HTTP :80
                       │
                       ▼
              ┌─────────────────┐
              │     AWS EC2     │
              │                 │
              │     Nginx       │
              │      :80        │
              │       │         │
              │       ▼         │
              │     Go API      │
              │     :8080       │
              │       │         │
              │       ▼         │
              │   PostgreSQL    │
              │     :5432       │
              │                 │
              └─────────────────┘

The Go API and PostgreSQL database are kept internal to the Docker Compose network. Nginx is the public entry point.

Application

The backend is written in Go and exposes REST endpoints for a simple user-management API.

Endpoints

Method

Endpoint

Purpose

GET

/health

Application health check

GET

/db-health

Database connectivity check

POST

/users

Create a user

GET

/users

List users

GET

/users/{id}

Get a user

PUT

/users/{id}

Update a user

DELETE

/users/{id}

Delete a user

Validation

The API includes:

Required name and email validation

Maximum field length validation

Basic email-format validation

Invalid ID handling

Duplicate email conflict handling

JSON error responses

Appropriate HTTP status codes

Project Structure

containerlab/
├── .github/
│   └── workflows/
│       └── ci.yml
├── app/
│   ├── cmd/
│   │   └── api/
│   │       └── main.go
│   ├── go.mod
│   └── go.sum
├── terraform/
│   ├── main.tf
│   ├── variables.tf
│   └── terraform.tfvars.example
├── .dockerignore
├── .gitignore
├── Dockerfile
├── compose.yaml
└── README.md

Terraform state, local variables, .env, and the .terraform/ directory are intentionally excluded from Git.

Docker

The application uses a multi-stage Docker build.

Build stage

The Go application is compiled using the Go 1.25 image.

Runtime stage

The compiled binary is copied into a lightweight Alpine Linux 3.22 runtime image.

The runtime container:

Uses a non-root appuser

Exposes port 8080

Includes a Docker healthcheck

Installs only the runtime package needed for health checking

Uses upgraded Alpine packages

Build flow

Go Source
   │
   ▼
Go Builder Image
   │
   ▼
Compiled Binary
   │
   ▼
Alpine Runtime Image
   │
   ▼
Container

Docker Compose

The EC2 deployment runs three services:

nginx
api
db

Nginx

Public port: 80

Reverse proxies requests to the API

Receives external HTTP traffic

API

Internal container port: 8080

Uses the GHCR image

Waits for PostgreSQL health before startup

PostgreSQL

Internal container port: 5432

Uses a persistent Docker volume

Healthchecked with pg_isready

CI Pipeline

GitHub Actions runs the current CI workflow on pushes to main and pull requests targeting main.

Pipeline stages

Checkout
   ↓
Setup Go 1.25
   ↓
Download dependencies
   ↓
Run Go tests
   ↓
Build Go application
   ↓
Build Docker image
   ↓
Trivy vulnerability scan
   ↓
Login to GHCR
   ↓
Push Docker image

Images are pushed to:

ghcr.io/abhishekkumar2484/containerlab

The workflow publishes both:

latest
<commit-sha>

on pushes to main.

DevSecOps

Trivy is integrated directly into GitHub Actions.

The pipeline scans the built image for:

OS package vulnerabilities

Go library vulnerabilities

HIGH severity issues

CRITICAL severity issues

The CI scan is configured to fail the workflow when matching vulnerabilities are found, while ignoring unfixed vulnerabilities.

Security remediation performed

During development, a HIGH vulnerability was identified in:

golang.org/x/text

The dependency was upgraded from an older vulnerable version to v0.39.0.

The Alpine base image was also upgraded from Alpine 3.20 to Alpine 3.22, followed by:

apk update
apk upgrade

The final Trivy verification showed zero OS and Go-binary vulnerabilities for the final image.

AWS Infrastructure

The application is deployed in AWS ap-south-1.

Current infrastructure includes:

Default VPC

Public subnet

Internet Gateway

Route table

Security Group

EC2 instance (t3.micro)

Network flow

Internet
   │
   ▼
Security Group
   │
   ▼
Public EC2
   │
   ▼
Nginx :80
   │
   ▼
Internal API :8080
   │
   ▼
Internal PostgreSQL :5432

The public application entry point is Nginx on port 80.

The API and database are not intended to be directly exposed to the public internet.

Administrative access is restricted through the configured administrative CIDR.

Terraform / Infrastructure as Code

AWS infrastructure is managed with Terraform.

Terraform manages the current AWS resources including:

Default VPC

Subnet

Security Group

Internet Gateway

Route table

EC2 instance

Existing AWS resources were imported into Terraform state so that the existing environment could be managed as Infrastructure as Code.

Terraform workflow

terraform init
terraform plan
terraform apply

The current infrastructure has been reconciled successfully and:

terraform plan

returns:

No changes

This confirms that the Terraform configuration and the managed AWS resources are currently synchronized.

Environment Configuration

Local and server configuration uses environment variables rather than committing credentials into source control.

Example:

POSTGRES_USER=containerlab
POSTGRES_PASSWORD=devpassword
POSTGRES_DB=containerlab

Sensitive local files such as .env and terraform.tfvars are ignored by Git.

The example password above is for the learning/development setup only and should be replaced with a strong secret in a real production environment.

Verification

Application health

curl -i http://<EC2_PUBLIC_IP>/health

Expected response:

HTTP/1.1 200 OK

{"status":"ok"}

Docker services

On the EC2 instance:

docker compose ps

The expected services are:

containerlab-nginx   Up
containerlab-api     Up (healthy)
containerlab-db      Up (healthy)

Terraform

terraform plan

Expected:

No changes

Security Practices Implemented

Multi-stage Docker build

Non-root application user

Internal API and database networking

Nginx as the public reverse proxy

Security Group based network controls

Trivy vulnerability scanning in CI

Vulnerable dependency remediation

Updated Alpine base image

Gitignored secrets and Terraform state

Image scanning before publishing

Docker healthcheck for the API

Current Status

Completed

Go REST API

PostgreSQL integration

CRUD operations

API validation and error handling

Docker multi-stage build

Non-root container execution

Docker healthcheck

Docker Compose deployment

Nginx reverse proxy

GitHub Actions CI

Docker image build

Trivy security scanning

GHCR image publishing

AWS EC2 deployment

AWS networking configuration

Terraform Infrastructure as Code

Terraform state import/reconciliation

Public health endpoint verification

Final Terraform drift check

Current deployment flow

GitHub
   ↓
GitHub Actions
   ↓
Tests + Build + Trivy
   ↓
GHCR
   ↓
AWS EC2
   ↓
Docker Compose
   ↓
Nginx
   ↓
Go API
   ↓
PostgreSQL

Why This Project Matters

ContainerLab demonstrates practical DevOps responsibilities rather than only application development:

Building and packaging an application

Automating software validation

Adding security checks to CI

Managing container images

Provisioning and managing cloud infrastructure

Applying Infrastructure as Code

Configuring cloud networking and firewall rules

Running a multi-container application

Exposing services safely through a reverse proxy

Verifying real production-like deployment behavior

The project is designed as a hands-on demonstration of a complete path from source code to a live cloud-hosted application.

Technology Stack

Area

Technology

Language

Go 1.25

API

Go REST API

Database

PostgreSQL 16

Containers

Docker

Container orchestration (current)

Docker Compose

Reverse proxy

Nginx

CI

GitHub Actions

Container registry

GitHub Container Registry (GHCR)

Security scanning

Trivy

Cloud

AWS

Compute

EC2

Infrastructure as Code

Terraform

OS/runtime image

Alpine Linux 3.22

Author

Built as a hands-on DevOps / Cloud Engineering portfolio project by Abhishek Kumar.# containerslab
