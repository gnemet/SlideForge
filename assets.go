package SlideForge

import "embed"

//go:embed ui/templates/* ui/static/* resources/*
var EmbeddedAssets embed.FS
