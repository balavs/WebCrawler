package main

import (
    "fmt"
    //"strings"
    "log"
    "time"
    "os"
    "net/http"
    "net/url"

    "github.com/andybalholm/cascadia"
    "golang.org/x/net/html"
)

//Configuration constants
const numThreads int = 2 //Determines the number of concurrent crawlers
const crawlDepth int = 1 //Sets the crawl depth 
const crawlDelay time.Duration = 1000 //Time delay for crawlThread to move on to next link.
				      //In units of milliseconds.

//Structure for the required log information
//
type LogInfo struct {
	HostName string
	RespCode string
	UrlPath  string
	TStamp   time.Time
}

//Structure to store the crawled URL stats including the log info.
//
type CrawlStat struct {
	Crawled bool
	Depth int
	Running bool
	Log LogInfo
}

//Structure to store the URL that needs to be crawled along with
//their depth info relative to seed url.
type UrlInfo struct {
	UrlN  string
	Depth int
}

//This map variable stores the satisitics of the crawled URL and uses
//the URL as the index.
var crawlStatMap map[string]CrawlStat

//This map variable stores the URL to be crawled as the index and the
//depth at which this url was located as the value. 
var crawlListMap map[string]int

// Pre-compiled cascadia css selector patterns
var atag = cascadia.MustCompile("a")


//statThread() function is the control thread for the crawler and it 
//also does the bookeeping of the crawlStatMap and crawlListMap maps. 
//This thread blocks on appurl channel to receive the new links from 
//crawlThread, pushes them into crawlListMap and triggers a crawlThread 
//if one is available. This thread also blocks on the updurl channel to 
//receive the loginfo of the url that has been crawled, pushes this 
//info into crawlStatMap and triggers a crawlThread if more URLs are 
//available in crawlListMap.       
//
func statThread(crwlurl chan<- UrlInfo, appurl <-chan UrlInfo,
                updurl <-chan LogInfo, crwldone chan bool) {
	      //println("Started STAT Thread")
              var ap,up CrawlStat
              var crawlUrli UrlInfo
	      for {
		select {
		 case appendUrl := <-appurl:
			//Check for presence of the new URL in both
			//crawlStatMap and crawlListMap before adding
			//the new URL to crawlListMap 
			_,prsnts := crawlStatMap[appendUrl.UrlN]
			_,prsntl := crawlListMap[appendUrl.UrlN]
			if (!prsnts && !prsntl) {
	                  crawlListMap[appendUrl.UrlN] = appendUrl.Depth
			}
			//Pop out the next entry from crawlListMap
			for crawlUrl, dep := range crawlListMap {
				crawlUrli.UrlN = crawlUrl; crawlUrli.Depth = dep;
				break
			}
			//Non-blocking select on crwlurl channel to see if a 
			//crawlThread is available to crawl a new URL.IF available 
			//send the URL on crwlurl channel, remove the url from 
			//crawlListMap and add it to crawlStatMap
			select {
			case crwlurl <- crawlUrli:
				ap.Crawled = false; ap.Depth = crawlUrli.Depth; ap.Running = true
				ap.Log = LogInfo{}
				crawlStatMap[crawlUrli.UrlN] = ap
				delete(crawlListMap,crawlUrli.UrlN)
			default :
				//Do nothing. Just to make the select non-blocking
			}
		 case updateUrl := <-updurl:
			//Collect the loginfo into CrawlStat struct and 
			//update it in the crawlStatMap
                        depth := crawlListMap[updateUrl.UrlPath]
	                up.Crawled = true; up.Depth = depth; up.Running = false
			up.Log = updateUrl
	                crawlStatMap[updateUrl.UrlPath] = up
			//Check is any more URLs are available in crawlListMap and 
			//if not signal that crawling is complete on crwldone channel
	                crawlUrl, crawlPend := crawlPending()
	                if (!crawlPend) { //TODO Also check if any crawl is in progress for multiple crawl threads
			    crwldone <- true
	                    break
	                }
			//If there is new URL available send the URL on crwlurl 
			//channel, remove the url from crawlListMap and add it to crawlStatMap  
			ap.Crawled = false; ap.Depth =crawlUrl.Depth; ap.Running = true
			ap.Log = LogInfo{}
			crawlStatMap[crawlUrl.UrlN] = ap
			delete(crawlListMap,crawlUrl.UrlN)
	                crwlurl <- crawlUrl
		}
	      }
}

//crawlPending() function checks if the crawlListMap is empty and if
//it is not it pops out the first entry in the iterator and returns it.
//If it is empty it retutns a dummy url with the bool flag set to false
//indicating that the map is empty.
//
func crawlPending() (UrlInfo, bool) {
    //Check if the crawlListMap is not empty and then popout the first iterator
    if (len(crawlListMap) > 0) {
      for url, dep := range crawlListMap {
          //fmt.Printf("%s -> %t\n", url,crawled)
	    urli := UrlInfo{url,dep}
            return urli, true
      }
    }
    return UrlInfo{"done",0}, false
}

//crawlThread() blocks on crwlurl channel which is set by the statThread
//whenever a new URL is availble for crawling and crawlThread is available.
//This thread implements a coarse fairness in crawling by sleeping for 
//crawlDelay time and can be viewed as a rate limiter also.
//
func crawlThread(crwlurl <-chan UrlInfo, appurl chan<- UrlInfo, updurl chan<- LogInfo, i int) {
	      //fmt.Printf("Started CRAWL Thread %d\n", i)
	      for {
		select {
		 case crawlUrl := <-crwlurl:
			//fmt.Printf("Thread %d : %s\n",i,crawlUrl.UrlN)
			getUrlLinks(crawlUrl,appurl,updurl)
		}
		//fmt.Printf("Thread %d Sleeping\n",i)
	        time.Sleep(crawlDelay * time.Millisecond)
	      }
}


//getUrlLinks() function takes in a URlInfo struct(geturli) to request
//the specified URL and parse the received page if the depth is within
//the specified limits. It parses out all the links in the and sends each
//of them on the appurl channel to the statThread. It returns to the
//crawlThread after all the links are exhausted or if the current url
//depth matches the set depth limit. Wile returning it send the log info 
//for the current url on the updurl channel to the statThread. 
//
func getUrlLinks(geturli UrlInfo, appurl chan<- UrlInfo, updurl chan<- LogInfo) {

	var urlLog LogInfo

	//println(geturli.UrlN) 
	//Do a Get request to get the URL html content
        resp, err := http.Get(geturli.UrlN)
	if err != nil {
		log.Fatal(err)
	}

        //Get all the log info for the current URL into urlLog struct
	uri,err := url.Parse(geturli.UrlN)
	urlLog.HostName = uri.Host;
	urlLog.UrlPath = geturli.UrlN
	urlLog.RespCode = resp.Status;
	urlLog.TStamp = time.Now()

	//Only if the current URL depth is less than the crawlDepth config value,
	//parse the page to get the links.
	if (geturli.Depth < crawlDepth) {

		//Generate the parse tree for the html
		doc, err := html.Parse(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		//Find all the "href" links and push each one of them into the appurl
		//channel along with the incremented depth value. If the link is same
		//as the current link then dont send it on the channel.
		for _,link := range atag.MatchAll(doc) {
			fullurl := urlJoin(geturli.UrlN, getAttrVal(link,"href"))
			//??println(fullurl)

			if (fullurl != geturli.UrlN) {
			   appurl <- UrlInfo{fullurl, geturli.Depth+1}
			}
		}
	}

	//Send the loginfo for the current url on updurl channnel and return
        updurl <- urlLog
	return
}

//urlJoin() function takes base url and reference url and returns an
//absolute version of the reference url.
// 
func urlJoin(base, href string) string {
	//Parse the raw href url string to url struct
	uri, err := url.Parse(href)
	if err != nil {
		return ""
	}
	//To avoid the "javascript" and other unsupported schemes. Just ignore them
	// and return the base url itself.
        if ((uri.Scheme != "http") && (uri.Scheme != "https") && (uri.Scheme != "")) {
	  return base
        }

	//Parse the raw base url string to url struct
	baseUrl, err := url.Parse(base)
	if err != nil {
		return ""
	}
	//??println(uri.String())
	//Prepends the uri with baseURL base if uri is a realtive URL
	uri = baseUrl.ResolveReference(uri)
	return uri.String()
}

//getAttrVal() function iterates through all the attributes in the
//"ele" Node and returns the value of the first attribute that is
//of type "attrType". Use to get the "href" value in <a> tags.
//
func getAttrVal(ele *html.Node, attrType string) string {
		for _, a := range ele.Attr {
			if a.Key == attrType {
			    return a.Val
			}
		}
                return ""
}


func main() {

	//Read the seed URL input
	if(len(os.Args) < 2) {
	  log.Fatal("Please enter the seed URL Argument to the command line")
	}
	seedUrl := os.Args[1]

        //Log file creation
	f,_ := os.Create("webcrawl.log")
	//Check err
	defer f.Close()

	//Initialize the Map data structures
	//
	crawlStatMap = make(map[string]CrawlStat)
	crawlListMap = make(map[string]int)

        //Create channels:
	//crwlurl - channel for communicating the URL/depth to crawlThread
	//appurl - channel for communicating the URL/depth to statThread
	//updurl - channel for communicating the URL log info to statThread
	//crwldone - channel for communicating that all links have ben crawled to the main() from statThread
	crwlurl   := make(chan UrlInfo)
	appurl    := make(chan UrlInfo)
	updurl    := make(chan LogInfo)
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

	//Write out the log
	//
	for k,v := range crawlStatMap {
		fmt.Printf("%s -> %t -> %v\n", k,v.Crawled,v.Log)
	        f.WriteString(v.Log.HostName+","+v.Log.TStamp.Format(time.UnixDate)+","+v.Log.UrlPath+","+v.Log.RespCode+"\n")
	        f.Sync()
	}
}

//Error Handling
//Test Cases
//?Fine grain fairness
//HTTP request timeout handling?
//#Cmd line input
//#Parameterizing
//#File writing
//#Code align and beautificiton and comments
