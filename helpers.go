package main

import (
	"fmt"
	"os"
	"strings"
)

func sliceToMap(s []string, d string) (map[string]string, error) {
	vars := map[string]string{}
	for _, pair := range s {
		splits := strings.SplitN(pair, d, 2)
		if len(splits) == 2 {
			vars[splits[0]] = splits[1]
		} else {
			return vars, fmt.Errorf("variable '%s' could not be split by %s", pair, d)
		}
	}
	return vars, nil
}

func info(i string) {
	fmt.Fprintf(os.Stderr, "%s\n", i)
}

func exitOnErr(errs ...error) {
	errNotNil := false
	for _, err := range errs {
		if err == nil {
			continue
		}
		errNotNil = true
		fmt.Fprintf(os.Stderr, "ERROR: %s", err.Error())
	}
	if errNotNil {
		fmt.Print("\n")
		os.Exit(-1)
	}
}
