package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"parabellum.crawler/internal/crawler"
)

const pathToEnvFile = ".env"

const (
	TypeString = iota
	TypeInt
	TypeTimeSecond
)

//TestFilters represents how endpoints should be filtered between tests
var TestsFilters = map[string][crawler.NumOfBodyParams]bool{
	"SQLI-check": fillTestFilter("SQLi", true, false),
	"BA-check":   fillTestFilter("Broken auth", true, false),
	"XSS-check":  fillTestFilter("XSS", true, true),
	"LFI-check":  fillTestFilter("LFI", false, true),
}

func fillTestFilter(testName string, filters ...bool) [crawler.NumOfBodyParams]bool {
	if len(filters) != crawler.NumOfBodyParams {
		log.Panicln("Wrong filters number for test", testName)
	}

	var result [crawler.NumOfBodyParams]bool
	result[crawler.HasFormTag] = filters[crawler.HasFormTag]
	result[crawler.HasQueryParameter] = filters[crawler.HasQueryParameter]

	return result
}

//EnvVarOfType returns environment variable converted to a given type
func EnvVarOfType(varName string, varType int) any {
	strVal := os.Getenv(varName)
	switch varType {
	case TypeString:
		return strVal
	case TypeInt:
		res, _ := strconv.Atoi(strVal)

		return res
	case TypeTimeSecond:
		res, _ := strconv.Atoi(strVal)

		return time.Duration(res) * time.Second
	}

	return nil
}
