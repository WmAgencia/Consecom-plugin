package main

import "embed"

// CSS and JavaScript shipped within the Windows executable.
//go:embed chat_ui.css chat_ui.js
var chatAssets embed.FS
