package main

import (
	"fmt"
	"time"

	"github.com/nathants/go-webkit"
)

func main() {
	debug := true
	w := webkit.New(debug)
	defer w.Destroy()
	w.SetTitle("go-webkit")
	w.SetSize(800, 600, webkit.HintNone)
	w.Navigate("http://google.com")
	err := w.Bind("result", func(jsonVal string) { fmt.Println(jsonVal) })
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			selector := "a"
			attr := "href"
			w.Eval(fmt.Sprintf("result(JSON.stringify([...document.querySelectorAll(\"%s\")].map(x => x.%s)))", selector, attr))
			time.Sleep(1000 * time.Millisecond)
		}
	}()
	w.Run()
}
