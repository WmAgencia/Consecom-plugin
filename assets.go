package main

import "embed"

//go:embed logo.png ConsecomMCP.ico
var art embed.FS
var iconPNG, _ = art.ReadFile("logo.png")
var iconICO, _ = art.ReadFile("ConsecomMCP.ico")
