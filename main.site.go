package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hornbill/pb"
)

//loadSites
func loadSites() {
	pageSize := 100
	rowStart := 0

	hornbillImport.SetParam("rowstart", "0")
	hornbillImport.SetParam("limit", "1")
	hornbillImport.SetParam("orderByField", "h_site_name")
	hornbillImport.SetParam("orderByWay", "ascending")

	RespBody, xmlmcErr := hornbillImport.Invoke("apps/com.hornbill.core", "getSitesList")
	var JSONResp xmlmcSiteResponse
	if xmlmcErr != nil {
		logger(4, "Unable to Query Sites List "+fmt.Sprintf("%s", xmlmcErr), false, true)
		return
	}

	err := json.Unmarshal([]byte(RespBody), &JSONResp)
	if err != nil {
		logger(4, "Unable to Read Sites List "+err.Error(), false, true)
		return
	}
	if JSONResp.State.Error != "" {
		logger(4, "Unable to Query Groups List "+JSONResp.State.Error, false, true)
		return
	}
	count, _ := strconv.Atoi(JSONResp.Params.Count)
	//-- Load Results in pages of pageSize
	bar := pb.StartNew(count)
	for rowStart < count {
		logger(1, "Loading Site List Offset: "+fmt.Sprintf("%d", rowStart)+"\n", false, true)
		loopCount := 0

		hornbillImport.SetParam("rowstart", strconv.Itoa(rowStart))
		hornbillImport.SetParam("limit", strconv.Itoa(pageSize))
		hornbillImport.SetParam("orderByField", "h_site_name")
		hornbillImport.SetParam("orderByWay", "ascending")

		RespBody, xmlmcErr = hornbillImport.Invoke("apps/com.hornbill.core", "getSitesList")
		if xmlmcErr != nil {
			logger(4, "Unable to Query Sites List "+fmt.Sprintf("%s", xmlmcErr), false, true)
			return
		}

		err = json.Unmarshal([]byte(RespBody), &JSONResp)
		if err != nil {
			logger(4, "Unable to Read Sites List "+err.Error(), false, true)
			return
		}

		if JSONResp.State.Error != "" {
			logger(4, "Unable to Query Groups List "+JSONResp.State.Error, false, true)
			return
		}

		if JSONResp.Params.Sites[7] == 91 { // [
			var JSONSites xmlmcSitesReader
			err = json.Unmarshal([]byte(JSONResp.Params.Sites), &JSONSites)
			if err != nil {
				logger(4, "Unable to Read Sites "+err.Error(), false, true)
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
				loopCount++
			}
		} else {
			var JSONSites xmlmcIndySite
			err = json.Unmarshal([]byte(JSONResp.Params.Sites), &JSONSites)
			if err != nil {
				logger(4, "Unable to Read Site "+err.Error(), false, true)
				return
			}
			var newSiteForCache siteListStruct
			newSiteForCache.SiteID, _ = strconv.Atoi(JSONSites.Row.ID)
			newSiteForCache.SiteName = JSONSites.Row.Name
			name := []siteListStruct{newSiteForCache}
			mutexSite.Lock()
			Sites = append(Sites, name...)
			mutexSite.Unlock()
			loopCount++
		}
		// Add 100
		bar.Add(loopCount)
		rowStart += loopCount
	}
	bar.FinishPrint("Sites Loaded  \n")
	logger(1, "Sites Loaded: "+strconv.Itoa(len(Sites)), false, true)
}

func getSiteID(u map[string]interface{}, buffer *bytes.Buffer) (siteID int, siteName string) {
	siteNameMapping := fmt.Sprintf("%v", SQLImportConf.AssetGenericFieldMapping["h_site"])
	siteName = getFieldValue("h_site", siteNameMapping, u, buffer)
	if siteName != "" && siteName != "__clear__" {
		mutexSite.Lock()
		//-- Check if in Cache
		for _, site := range Sites {
			if strings.EqualFold(site.SiteName, siteName) {
				siteID = site.SiteID
				break
			}
		}
		mutexSite.Unlock()
	}
	debugLog(buffer, "Site Mapping:", siteNameMapping, ":", siteName, ":", strconv.Itoa(siteID))
	return
}
