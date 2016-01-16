package main

import (
    "fmt"
    "log"
    "time"
    "os"
    "net/url"
)

//Configuration constants
const numThreads int = 2 //Determines the number of concurrent crawlers
const crawlDepth int = 3 //Sets the crawl depth 
const crawlDelay time.Duration = 1000 //Time delay for crawlThread to move on to next link.
				      //In units of milliseconds.
func main() {

	//Read the seed URL input
	if(len(os.Args) < 2) {
	  log.Fatal("Please enter the seed URL Argument to the command line")
	}
	seedUrl := os.Args[1]
	//Parse the seed URL to make sure that a scheme has been added and
	//if not add the http scheme 
	seedUrli, err := url.Parse(seedUrl)
	if(err != nil) {
	  log.Fatal("Please check that the URL format entered is a valid one")
	}
	if (seedUrli.Scheme == ""){
	   seedUrli.Scheme = "http"
	   println("Adding http scheme to seed url")
	   seedUrl = seedUrli.String()
	   fmt.Printf("Seed Url: %s",seedUrl)
	}

        //Create channels:
	//crwlurl - channel for communicating the URL/depth to crawlThread
	//appurl - channel for communicating the URL/depth to statThread
	//updurl - channel for communicating the URL log info to statThread
	//crwldone - channel for communicating that all links have been 
	//           crawled to the main() from statThread
	crwlurl   := make(chan UrlInfo,numThreads)
	appurl    := make(chan UrlInfo,numThreads)
	updurl    := make(chan LogInfo,numThreads)
	crwldone  := make(chan bool)

        // Spawn the stats collection thread
	go statThread(crwlurl,appurl,updurl,crwldone)

        // Spawn the crawler threads - the number of threads can be
	// coonfigure with the constant numThreads
	for i := 0; i < numThreads; i++ {
		go crawlThread(crwlurl,appurl, updurl, i)
	}

        // Send the seed url with depth set to appurl channel
	appurl <- UrlInfo{seedUrl,0}

	//Block until all the crawls upto to the specified depth are completed.
	<-crwldone

	//Print the done message
	fmt.Printf("Crawl done  for URL %s upto a depth of %d",seedUrl,crawlDepth )
}
