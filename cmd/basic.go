package main

import (
	"github.com/nathants/go-webkit"
)

func main() {
	debug := true
	w := webkit.New(debug)
	defer w.Destroy()
	w.SetTitle("go-webkit")
	w.SetSize(800, 600, webkit.HintNone)
	w.Navigate("https://google.com")
	w.Run()
}
