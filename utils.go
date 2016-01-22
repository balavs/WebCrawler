//Helper functions for the webcrawler main() function.
//
package main

import (
    "time"
    "os"
    "log"
    "net/url"
    "net/http"
    "golang.org/x/net/html"
    "github.com/andybalholm/cascadia"
)

//Structure for the required log information
//
type LogInfo struct {
	HostName string    //Host name
	RespCode string    //Response code for the HTTP request to the URL
	UrlPath  string    //Full URL path
	TStamp   time.Time //Date and time of visit
}

//Structure to store the crawled URL stats including the log info.
//
type CrawlStat struct {
	Crawled bool   //Flag to indicate that the crawling is done for a URL
	Depth int      //Indicates the depth at which the URL was found
	Running bool   //Flag to indicate is the parsing of the URL is in 
		       //progress.(May be redundant with Crawled? Check and remove)
	Log LogInfo    //Log info for the URL after it has been crawled
}

//Structure to store the URL that needs to be crawled along with
//their depth info relative to seed url.
type UrlInfo struct {
	UrlN  string  //URL path
	Depth int     //Depth at which this URL was found
}

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
              //This map variable stores the satisitics of the crawled URL and uses
              //the URL as the index.
              var crawlStatMap map[string]CrawlStat

	      //This map variable stores the URL to be crawled as the index and the
	      //depth at which this url was located as the value. 
	      var crawlListMap map[string]int

	      //Initialize the Map data structures
	      //
	      crawlStatMap = make(map[string]CrawlStat)
	      crawlListMap = make(map[string]int)

              //This global var keeps track of urls still pending to be crawled in the crawlStatMap
              var pendCrawlCnt int = 0

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
				pendCrawlCnt = mvUrlMaps(crawlUrli,pendCrawlCnt,crawlStatMap,crawlListMap)
			default :
				//Do nothing. Just to make the select non-blocking
			}
		 case updateUrl := <-updurl:
			//Collect the loginfo into CrawlStat struct and 
			//update it in the crawlStatMap
                        up := crawlStatMap[updateUrl.UrlPath]
	                up.Crawled = true; up.Running = false
			up.Log = updateUrl
	                crawlStatMap[updateUrl.UrlPath] = up
			pendCrawlCnt = pendCrawlCnt - 1
			//Check is any more URLs are available in crawlListMap and 
			//if not signal that crawling is complete on crwldone channel 
			//Also check if any crawl is in progress for multiple crawl
			//threads using pendCrawlCnt
	                crawlUrl, crawlPend := crawlPending(crawlListMap )
	                if (!crawlPend && pendCrawlCnt==0) {
			    writeLog(crawlStatMap)
			    crwldone <- true
	                    break
	                }
			//If there is new URL available send the URL on crwlurl 
			//channel, remove the url from crawlListMap and add it to crawlStatMap  
	                if (crawlPend) {
			  select {
				case crwlurl <- crawlUrl:
					pendCrawlCnt = mvUrlMaps(crawlUrl,pendCrawlCnt,crawlStatMap,crawlListMap)
					//fmt.Printf("UPD:-%d\n",pendCrawlCnt)
				default :
					//Do nothing. Just to make the select non-blocking
			  }
			}
		 default :
				//Do nothing. Just to make the select non-blocking
		}
	      }
}

//crawlPending() function checks if the crawlListMap is empty and if
//it is not it pops out the first entry in the iterator and returns it.
//If it is empty it retutns a dummy url with the bool flag set to false
//indicating that the map is empty.
//
func crawlPending(listMap map[string]int) (UrlInfo, bool) {
    //Check if the listMap is not empty and then popout the first iterator
    if (len(listMap) > 0) {
      for url, dep := range listMap {
          //fmt.Printf("%s -> %t\n", url,crawled)
	    urli := UrlInfo{url,dep}
            return urli, true
      }
    }
    return UrlInfo{"done",0}, false
}

func writeLog(crawlStatMap map[string]CrawlStat) {
        //Log file creation
	f,err := os.Create("webcrawl.log")
	//On file open error, print the message and exit
	if (err != nil) {
	  log.Fatal(err)
	}
	defer f.Close()

	//Write out the log
	//
	for _,v := range crawlStatMap {
		//fmt.Printf("%s -> %t -> %v\n", k,v.Crawled,v.Log)
	        f.WriteString(v.Log.HostName+","+v.Log.TStamp.Format(time.UnixDate)+","+v.Log.UrlPath+","+v.Log.RespCode+"\n")
	        f.Sync()
	}
}

//mvUrlMaps() moves the URL from crawlListMap to crawlStatMap and 
//increments the pendCrawlCnt to keepo track of pending crawls 
//
func mvUrlMaps(crawlUrl UrlInfo,pendCrawlCnt int,
	       crawlStatMap map[string]CrawlStat,crawlListMap map[string]int) int {
        var ap CrawlStat
	ap.Crawled = false; ap.Depth = crawlUrl.Depth; ap.Running = true
	ap.Log = LogInfo{}
	crawlStatMap[crawlUrl.UrlN] = ap
	pendCrawlCnt = pendCrawlCnt + 1
	delete(crawlListMap,crawlUrl.UrlN)
	return pendCrawlCnt
}


//crawlThread() blocks on crwlurl channel which is set by the statThread
//whenever a new URL is availble for crawling and crawlThread is available.
//This thread implements a coarse fairness in crawling by sleeping for 
//crawlDelay time and can be viewed as a rate limiter also.
//
func crawlThread(crwlurl <-chan UrlInfo, appurl chan<- UrlInfo, updurl chan<- LogInfo, i int) {
	      //fmt.Printf("Started CRAWL Thread %d\n", i)
	      var prevHst string = "start"
	      for {
		select {
		 case crawlUrl := <-crwlurl:
			//Delay the http request if the thread had 
			//requested to the same Host in the previous iteration
			crawlUrli,_ := url.Parse(crawlUrl.UrlN)
			crawlHst := crawlUrli.Host
			if (prevHst != "start" && prevHst == crawlHst) {
			  //fmt.Printf("%s-%s Delaying\n",prevHst,crawlHst)
			  time.Sleep(crawlDelay * time.Millisecond)
			} else {
			  //fmt.Printf("%s-%s No Delay\n",prevHst,crawlHst)
			}
			prevHst = crawlHst
			//fmt.Printf("Thread %d : %s\n",i,crawlUrl.UrlN)
			getUrlLinks(crawlUrl,appurl,updurl)
		}
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

        //Get the URL path into log info
	urlLog.UrlPath = geturli.UrlN

	//Parse the url string to get the url struct
	uri,err := url.Parse(geturli.UrlN)
	//On parse error (mostly due to badly formed url), set the log info
	//and return so that the thread can receive the next URL for crawling
	if err != nil {
	  println("Invald URl")
	  urlLog.RespCode = "Invalid URL"
	  urlLog.TStamp = time.Now()
	  //Send the loginfo for the current url on updurl channnel and return
          updurl <- urlLog
	  return
	}
        //Set the Hostname info for the current URL into urlLog struct
	urlLog.HostName = uri.Host

	//println(geturli.UrlN) 
	//Do a Get request to get the URL html content
        resp, err := http.Get(geturli.UrlN)
        //Set the request timestamp info for the current URL into urlLog struct
	urlLog.TStamp = time.Now()

	//On request error, set the response code to the error message received
	//and return so that the thread can receive the next URL for crawling
	if err != nil {
	  //log.Fatal(err)
	  urlLog.RespCode = err.Error()
	  //Send the loginfo for the current url on updurl channnel and return
          updurl <- urlLog
	  return
	} else {
	//If there is no error in response, set the response status as the 
	//response code and continue parsing the page contents
	  urlLog.RespCode = resp.Status
	}
	defer resp.Body.Close()

	//Only if the current URL depth is less than the crawlDepth config value,
	//parse the page to get the links.
	if (geturli.Depth < crawlDepth) {

		//Generate the parse tree for the html
		doc, err := html.Parse(resp.Body)
		// On HTML parse error, add the error message to the Url Response code
		if err != nil {
			//log.Fatal(err)
			urlLog.RespCode = urlLog.RespCode + "HTML Parse Error"
			//Send the loginfo for the current url on updurl channnel and return
			updurl <- urlLog
			return
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
	//On a malformed href url, just return the href back so that the
	//url parse error can be recorded when it is crawled
	if err != nil {
		return href
	}
	//To avoid the "javascript" and other unsupported schemes. Just ignore them
	// and return the base url itself.
        if ((uri.Scheme != "http") && (uri.Scheme != "https") && (uri.Scheme != "")) {
	  return base
        }

	//Parse the raw base url string to url struct
	baseUrl, err := url.Parse(base)
	//This error will not happen as this parse is already done in the calling
	//function once
	if err != nil {
		return base
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

