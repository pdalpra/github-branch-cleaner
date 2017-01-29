package main

import (
	"fmt"
	"os"
	"strings"
)

func logAndExitIfError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func requiredEnvVar(varName string) string {
	v := os.Getenv(varName)
	if v == "" {
		logAndExitIfError(fmt.Errorf("%s environment variable is not defined", varName))
	}
	return v
}

func optionalEnvVar(varName string, defaultValue string) string {
	v := os.Getenv(varName)
	if v == "" {
		return defaultValue
	}
	return v
}

func parseExclusionList(envVar string) []string {
	if envVar == "" {
		return []string{}
	}
	return strings.Split(envVar, ",")
}
