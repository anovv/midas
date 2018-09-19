package main

import (
	. "midas/eyes"
)

func main() {
	SetupEye()
	defer CleanupEye()
}