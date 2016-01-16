# WebCrawler
GoLang implementation of WebCrawler.This webcrawler will take a "seed url" from the command line argument and will crawl the links from that url recursively upto a depth of 3. After the crawling is complete, it will log all the information on all the links visited in a log file "webcrawl.log".

There are 3 configurable parameters in webcrawler.go that can be changed to configure the webcrawler:

const numThreads int = 2 //Determines the number of concurrent crawlers. Default is set to 2 but can be increased depending on the resource available.
const crawlDepth int = 3 //Sets the crawl depth. This is set to 3 based on the requirement but can be changed if required. 
const crawlDelay time.Duration = 1000 //Time delay for crawlThread to move on to next link in units of milliseconds. Default is 1 second and can be changed as required.

More info can be found in the design document.

#Dependencies:
Installation of GoLang and Go environment setup is a pre-requisite.
Installation of the following external go libraries is required:

go get github.com/andybalholm/cascadia

go get golang.org/x/net/html

#Command to run the program directly
go run utils.go webcrawler.go \<http://\<full.url.path\>

#Command to create the binary
From the WebCrawler directory:

go install

This will create the binary WebCrawler(unix)/WebCrawler.exe(Windows) in $GOBIN/%GOBIN% dir. The binary can be executed as follows:

Unix    : $GOBIN/WebCrawler \<http://full.url.path\>

Windows : %GOBIN%/WebCrawler.exe \<http://full.url.path\>

