package main

import (
	"github.com/nathants/go-webkit"
)

func main() {

	debug := true
	w := webkit.New(debug)
	defer w.Destroy()
	w.SetTitle("Minimal webview example")
	w.SetSize(800, 600, webkit.HintNone)
	w.Navigate("https://en.m.wikipedia.org/wiki/Main_Page")
	w.Run()

}
