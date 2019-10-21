// +build !linux

package command

import "github.com/gobuffalo/packr"

// Initialize a packr box which will cotain all scripts.
var scriptBox = packr.Box{}
var scriptsAvailable = false
