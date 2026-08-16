#!/bin/sh
set -eu
go test ./...
go build -o /tmp/flagged-commerce-cutover ./cmd/checkout-service
