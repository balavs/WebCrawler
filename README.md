# WebCrawler
GoLang implementation of WebCrawler.This webcrawler will take a "seed url" from the command line argument and will crawl the links from that url recursively upto a depth of 3. After the crawling is complete, it will log all the information on all the links visited in a log file "webcrawl.log".

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

Unix    : $GOBIN/WebCrawler\ <http://full.url.path\>
Windows : %GOBIN%/WebCrawler.exe \<http://full.url.path\>

