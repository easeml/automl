package command

import (
	"github.com/gobuffalo/packr/v2"
)

// Initialize a packr box which will cotain all scripts.
var scriptBox = packr.New("scripts","../../dev/scripts/linux")
var scriptsAvailable = true
