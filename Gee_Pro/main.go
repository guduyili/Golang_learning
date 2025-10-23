// package main

// import (
// 	"fmt"
// 	"log"
// 	"net/http"
// )

// func main() {
// 	http.HandleFunc("/", indexHandler)
// 	http.HandleFunc("/hello", helloHandler)
// 	log.Fatal(http.ListenAndServe(":8090", nil))
// }

// func indexHandler(w http.ResponseWriter, req *http.Request) {
// 	fmt.Fprintf(w, "URL.Path = %q\n", req.URL.Path)
// }

// func helloHandler(w http.ResponseWriter, req *http.Request) {
// 	for k, v := range req.Header {
// 		fmt.Fprintf(w, "Header[%q] = %q\n", k, v)
// 	}
// }

// package main

// import (
// 	"fmt"
// 	"net/http"
// 	"log"
// )

// type Engine struct{}

// func (engine *Engine)ServerHTTP(w http.ResponseWritr, req *http.Request){
// 	switch req.URL.Path{
// 	case: "/":
// 	// fmt.Fprintf(w, "URL.Path = %q\n", req.URL.Path)
// 	fmt.Fprintf(w)

// 	default:
// 	fmt.Fprintf(w, "404 NOT FOUND %s\n", req.URL)
// }
// }

package main

import (
	"fmt"
	"net/http"

	"gee"
)

func main() {
	r := gee.New()
	r.GET("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "URL.Path = %q\n", req.URL.Path)
	})

	r.GET("/hello", func(w http.ResponseWriter, req *http.Request) {
		for k, v := range req.Header {
			fmt.Fprintf(w, "Header[%q] = %q\n", k, v)
		}
	})

	r.Run(":3000")
}
