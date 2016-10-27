package main

import (
	unrender "github.com/unrolled/render"
)

var render *unrender.Render

func initGlobals() {
	render = unrender.New(unrender.Options{
		IndentJSON: true,
		Layout:     "layout",
	})
}
