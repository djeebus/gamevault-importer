package main

import (
	"fmt"

	"gamevaultimporter/cmd"
)

func main() {
	if err := cmd.Cmd(); err != nil {
		fmt.Println(err)
	}
}
