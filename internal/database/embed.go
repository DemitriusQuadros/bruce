package database

import _ "embed"

// Schema contains the DDL executed on startup to create tables if they don't exist.
//
//go:embed schema.sql
var Schema string
