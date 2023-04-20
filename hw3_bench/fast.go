package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

//https://vostbur.github.io/posts/using-easyjson-with-glonag/
//https://transform.tools/json-to-go

type User struct {
	Browsers []string `json:"browsers"`
	Company  string   `json:"company"`
	Country  string   `json:"country"`
	Email    string   `json:"email"`
	Job      string   `json:"job"`
	Name     string   `json:"name"`
	Phone    string   `json:"phone"`
}

func FastSearch(out io.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			panic(err)
		}
	}()

	scanner := bufio.NewScanner(file)
	// TL;DR map[]struct{} is 5% faster in time and 10% less memory consumption comparing to map[]bool when it comes to a big Set.
	// https://itnext.io/set-in-go-map-bool-and-map-struct-performance-comparison-5315b4b107b
	seenBrowsers := make(map[string]struct{}, 0)

	uniqueBrowsers := 0
	foundUsers := ""

	i := -1
	for scanner.Scan() {
		i++
		line := scanner.Bytes()

		user := User{}
		err := user.UnmarshalJSON(line)
		if err != nil {
			panic(err)
		}

		isAndroid := false
		isMSIE := false

		for _, browser := range user.Browsers {

			if strings.Contains(browser, "MSIE") {
				isMSIE = true
				notSeenBefore := true

				if _, brwSeen := seenBrowsers[browser]; brwSeen {
					notSeenBefore = false
				}

				if notSeenBefore {
					// log.Printf("SLOW New browser: %s, first seen: %s", browser, user["name"])
					seenBrowsers[browser] = struct{}{}
					uniqueBrowsers++
				}
			}

			if strings.Contains(browser, "Android") {
				isAndroid = true
				notSeenBefore := true
				if _, brwSeen := seenBrowsers[browser]; brwSeen {
					notSeenBefore = false
				}

				if notSeenBefore {
					// log.Printf("SLOW New browser: %s, first seen: %s", browser, user["name"])
					seenBrowsers[browser] = struct{}{}
					uniqueBrowsers++
				}

			}

		}
		if isAndroid && isMSIE {
			// log.Println("Android and MSIE user:", user["name"], user["email"])
			email := strings.ReplaceAll(user.Email, "@", " [at] ")
			foundUsers += "[" + strconv.Itoa(i) + "] " + user.Name + " <" + email + ">\n"
		}

	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
	fmt.Fprintln(out, "found users:\n"+foundUsers)
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))
}
