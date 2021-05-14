package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	apiLib "github.com/hornbill/goApiLib"
	"github.com/hornbill/pb"
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
			break
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
	espXmlmc.SetParam("application", "com.hornbill.core")
	espXmlmc.SetParam("entity", "Site")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("column", "h_site_name")
	espXmlmc.SetParam("value", siteName)
	//espXmlmc.SetParam("h_site_name", siteName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")

	var XMLSTRING = espXmlmc.GetParam()
	XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords2")
	if xmlmcErr != nil {
		logger(4, "API Call failed when Searching Site:"+fmt.Sprintf("%v", xmlmcErr), false)
		logger(1, "API XML: "+XMLSTRING, false)
	}
	var xmlRespon xmlmcSiteListResponse

	err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
	if err != nil {
		logger(3, "Unable to Search for Site: "+fmt.Sprintf("%v", err), true)
		logger(1, "API XML: "+XMLSTRING, false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(3, "Unable to Search for Site: "+xmlRespon.State.ErrorRet, true)
			logger(1, "API XML: "+XMLSTRING, false)
		} else {
			//-- Check Response
			if xmlRespon.Params.RowData.Row.SiteName != "" {
				if strings.EqualFold(xmlRespon.Params.RowData.Row.SiteName, siteName) {
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

//loadSites
type xmlmcSiteResponse struct {
	Params struct {
		Sites string `json:"sites"`
		Count string `json:"count"`
	} `json:"params"`
	State stateJSONStruct `json:"state"`
}

type xmlmcSitesReader struct {
	Row []struct {
		ID   string `json:"h_id"`
		Name string `json:"h_site_name"`
	} `json:"row"`
}
type xmlmcIndySite struct {
	Row struct {
		ID   string `json:"h_id"`
		Name string `json:"h_site_name"`
	} `json:"row"`
}

func loadSites() {
	pageSize := 25
	rowStart := 0

	hornbillImport.SetParam("rowstart", "0")
	hornbillImport.SetParam("limit", "1")
	hornbillImport.SetParam("orderByField", "h_site_name")
	hornbillImport.SetParam("orderByWay", "ascending")

	RespBody, xmlmcErr := hornbillImport.Invoke("apps/com.hornbill.core", "getSitesList")
	var JSONResp xmlmcSiteResponse
	if xmlmcErr != nil {
		logger(4, "Unable to Query Sites List "+fmt.Sprintf("%s", xmlmcErr), false)
		return
	}

	err := json.Unmarshal([]byte(RespBody), &JSONResp)
	if err != nil {
		logger(4, "Unable to Read Sites List "+fmt.Sprintf("%s", err), false)
		return
	}
	if JSONResp.State.Error != "" {
		logger(4, "Unable to Query Groups List "+JSONResp.State.Error, false)
		return
	}
	count, _ := strconv.Atoi(JSONResp.Params.Count)
	//-- Load Results in pages of pageSize
	bar := pb.StartNew(count)
	for rowStart < count {
		logger(1, "Loading Site List Offset: "+fmt.Sprintf("%d", rowStart)+"\n", false)
		loopCount := 0

		hornbillImport.SetParam("rowstart", strconv.Itoa(rowStart))
		hornbillImport.SetParam("limit", strconv.Itoa(pageSize))
		hornbillImport.SetParam("orderByField", "h_site_name")
		hornbillImport.SetParam("orderByWay", "ascending")

		RespBody, xmlmcErr = hornbillImport.Invoke("apps/com.hornbill.core", "getSitesList")
		//var JSONResp xmlmcSiteResponse
		if xmlmcErr != nil {
			logger(4, "Unable to Query Sites List "+fmt.Sprintf("%s", xmlmcErr), false)
			return
		}

		err = json.Unmarshal([]byte(RespBody), &JSONResp)
		if err != nil {
			logger(4, "Unable to Read Sites List "+fmt.Sprintf("%s", err), false)
			return
		}
		if JSONResp.State.Error != "" {
			logger(4, "Unable to Query Groups List "+JSONResp.State.Error, false)
			return
		}

		//fmt.Println(JSONResp.Params.Sites)
		if JSONResp.Params.Sites[7] == 91 { // [
			var JSONSites xmlmcSitesReader
			err = json.Unmarshal([]byte(JSONResp.Params.Sites), &JSONSites)
			if err != nil {
				logger(4, "Unable to Read Sites "+fmt.Sprintf("%s", err), false)
				return
			}

			//-- Push into Map

			for index := range JSONSites.Row {
				var newSiteForCache siteListStruct
				newSiteForCache.SiteID, _ = strconv.Atoi(JSONSites.Row[index].ID)
				newSiteForCache.SiteName = JSONSites.Row[index].Name
				name := []siteListStruct{newSiteForCache}
				mutexSite.Lock()
				Sites = append(Sites, name...)
				mutexSite.Unlock()
				//fmt.Println(JSONSites.Row[index].Name)
				loopCount++
			}
		} else {
			var JSONSites xmlmcIndySite
			err = json.Unmarshal([]byte(JSONResp.Params.Sites), &JSONSites)
			if err != nil {
				logger(4, "Unable to Read Site "+fmt.Sprintf("%s", err), false)
				return
			}
			var newSiteForCache siteListStruct
			newSiteForCache.SiteID, _ = strconv.Atoi(JSONSites.Row.ID)
			newSiteForCache.SiteName = JSONSites.Row.Name
			name := []siteListStruct{newSiteForCache}
			mutexSite.Lock()
			Sites = append(Sites, name...)
			mutexSite.Unlock()
			//fmt.Println(JSONSites.Row.Name)
			loopCount++
		}
		// Add 100
		bar.Add(loopCount)
		rowStart += loopCount
		//-- Check for empty result set

	}
	bar.FinishPrint("Sites Loaded  \n")
	//	fmt.Println(Sites)
}
