package main

import (
	"fmt"

	"github.com/moqsien/gvc/pkgs/vctrl"
)

func main() {
	fmt.Println("hello world")
	h := vctrl.NewGoVersion()
	// h.Run()
	h.ShowVersions("1")
}
