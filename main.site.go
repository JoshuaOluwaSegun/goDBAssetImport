package main

import (
	"encoding/xml"
	"fmt"
	"github.com/hornbill/goApiLib"
	"strings"
)

// siteInCache -- Function to check if passed-thorugh site name has been cached
// if so, pass back the Site ID
func siteInCache(siteName string) (bool, int) {
	boolReturn := false
	intReturn := 0
	mutexSite.Lock()
	//-- Check if in Cache
	for _, site := range Sites {
		if site.SiteName == siteName {
			boolReturn = true
			intReturn = site.SiteID
		}
	}
	mutexSite.Unlock()
	return boolReturn, intReturn
}

// seachSite -- Function to check if passed-through  site  name is on the instance
func searchSite(siteName string, espXmlmc *apiLib.XmlmcInstStruct) (bool, int) {
	boolReturn := false
	intReturn := 0
	//-- ESP Query for site
	espXmlmc.SetParam("entity", "Site")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("h_site_name", siteName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")

	var XMLSTRING = espXmlmc.GetParam()
	XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords")
	if xmlmcErr != nil {
		logger(4, "API Call failed when Searching Site:"+fmt.Sprintf("%v", xmlmcErr), false)
		logger(1, "API XML: "+fmt.Sprintf("%s", XMLSTRING), false)
	}
	var xmlRespon xmlmcSiteListResponse

	err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
	if err != nil {
		logger(3, "Unable to Search for Site: "+fmt.Sprintf("%v", err), true)
		logger(1, "API XML: "+fmt.Sprintf("%s", XMLSTRING), false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(3, "Unable to Search for Site: "+xmlRespon.State.ErrorRet, true)
			logger(1, "API XML: "+fmt.Sprintf("%s", XMLSTRING), false)
		} else {
			//-- Check Response
			if xmlRespon.Params.RowData.Row.SiteName != "" {
				if strings.ToLower(xmlRespon.Params.RowData.Row.SiteName) == strings.ToLower(siteName) {
					intReturn = xmlRespon.Params.RowData.Row.SiteID
					boolReturn = true
					//-- Add Site to Cache
					var newSiteForCache siteListStruct
					newSiteForCache.SiteID = intReturn
					newSiteForCache.SiteName = siteName
					name := []siteListStruct{newSiteForCache}
					mutexSite.Lock()
					Sites = append(Sites, name...)
					mutexSite.Unlock()
				}
			}
		}
	}
	return boolReturn, intReturn
}
