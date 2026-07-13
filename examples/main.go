// Command example scrapes a page through the ProxyHat gateway with Colly.
//
// Run it with gateway credentials in the environment:
//
//	PROXYHAT_USERNAME=... PROXYHAT_PASSWORD=... go run ./examples
//
// or let an API key auto-select an active sub-user:
//
//	PROXYHAT_API_KEY=... go run ./examples
package main

import (
	"fmt"
	"log"

	collyproxyhat "github.com/ProxyHatCom/colly-proxyhat"
	"github.com/gocolly/colly/v2"
)

func main() {
	// Rotating US residential IPs (a fresh IP per connection). Swap Country for
	// your target, add collyproxyhat.Sticky() to pin one IP, or
	// collyproxyhat.WithSOCKS5() for SOCKS5.
	rotator, err := collyproxyhat.New(
		collyproxyhat.Country("us"),
		// collyproxyhat.Region("california"),
		// collyproxyhat.City("new_york"),
		// collyproxyhat.Filter("high"),
		// collyproxyhat.Sticky(),
	)
	if err != nil {
		log.Fatal(err)
	}

	c := colly.NewCollector()
	c.SetProxyFunc(rotator.ProxyFunc())

	c.OnResponse(func(r *colly.Response) {
		fmt.Printf("%d %s (%d bytes)\n", r.StatusCode, r.Request.URL, len(r.Body))
	})
	c.OnError(func(r *colly.Response, err error) {
		log.Printf("error for %s: %v", r.Request.URL, err)
	})

	if err := c.Visit("https://httpbin.org/ip"); err != nil {
		log.Fatal(err)
	}
	c.Wait()
}
