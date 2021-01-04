package tests

import (
	"fmt"
	"regexp"
	"testing"
)

func TestMatchedUri(t *testing.T) {
	uri1 := "/_left"
	uri2 := "/index/_left"
	uri3 := "/.index/_left"
	uri4 := "/.index1,.index1,.index1/_left"
	uri5 := "/.index*/_left"
	uri6 := "/*/_left"

	var matched bool
	pattern := ".*/_left"

	matched, _ = regexp.MatchString(pattern, uri1)
	fmt.Println("case 1 : ", matched)
	matched, _ = regexp.MatchString(pattern, uri2)
	fmt.Println("case 2 : ", matched)
	matched, _ = regexp.MatchString(pattern, uri3)
	fmt.Println("case 3 : ", matched)
	matched, _ = regexp.MatchString(pattern, uri4)
	fmt.Println("case 4 : ", matched)
	matched, _ = regexp.MatchString(pattern, uri5)
	fmt.Println("case 5 : ", matched)
	matched, _ = regexp.MatchString(pattern, uri6)
	fmt.Println("case 6 : ", matched)

}


