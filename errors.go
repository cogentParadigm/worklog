package main

import "fmt"

func handleError(err error) bool {
	if err != nil {
		fmt.Print(err)
		return true
	}
	return false
}
