package main

import "github.com/docToolchain/Bausteinsicht/internal/model"

// lockModel takes the lock of the model file for a command that reads the model,
// changes it and writes it back, and returns the function that releases it. The
// command must hold it from before the read until after the write, otherwise
// parallel commands overwrite each other's change (#712).
func lockModel(modelPath string) (func(), error) {
	return model.LockModel(modelPath)
}
