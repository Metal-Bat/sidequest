package main

import "embed"

//go:embed templates/*.html migrations/*.sql static/sidequest.css static/vendor/* static/fonts/*
var resources embed.FS
