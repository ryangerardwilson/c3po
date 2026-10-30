package bundle

import "embed"

//go:embed courses/*.md
var CourseFiles embed.FS
