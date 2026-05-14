package main

import "embed"

//go:embed web_dist/*
var webFS embed.FS
