package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/cheggaaa/pb"
)

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
		logger(4, "Unable to Query Sites List "+xmlmcErr.Error(), false, true)
		return
	}
	err := json.Unmarshal([]byte(RespBody), &JSONResp)
	if err != nil {
		logger(4, "Unable to Read Sites List "+err.Error(), false, true)
		return
	}
	if JSONResp.State.Error != "" {
		logger(4, "Unable to Query Sites List "+JSONResp.State.Error, false, true)
		return
	}
	//-- Load Results in pages of pageSize
	count, _ := strconv.Atoi(JSONResp.Params.Count)
	bar := pb.StartNew(count)
	for rowStart < count {
		logger(3, "Loading Site List Offset: "+fmt.Sprintf("%d", rowStart)+"\n", false, true)
		loopCount := 0

		hornbillImport.SetParam("rowstart", strconv.Itoa(rowStart))
		hornbillImport.SetParam("limit", strconv.Itoa(pageSize))
		hornbillImport.SetParam("orderByField", "h_site_name")
		hornbillImport.SetParam("orderByWay", "ascending")

		RespBody, xmlmcErr = hornbillImport.Invoke("apps/com.hornbill.core", "getSitesList")
		if xmlmcErr != nil {
			logger(4, "Unable to Query Sites List "+xmlmcErr.Error(), false, true)
			return
		}

		err = json.Unmarshal([]byte(RespBody), &JSONResp)
		if err != nil {
			logger(4, "Unable to Read Sites List "+err.Error(), false, true)
			return
		}

		if JSONResp.State.Error != "" {
			logger(4, "Unable to Query Sites List "+JSONResp.State.Error, false, true)
			return
		}

		if count > 1 {
			var JSONSites siteRowMultiple
			err = json.Unmarshal([]byte(JSONResp.Params.Sites), &JSONSites)
			if err != nil {
				logger(4, "Unable to Unmarshal Sites "+err.Error(), false, true)
				return
			}

			//-- Push into Map
			for index := range JSONSites.Row {
				var newSiteForCache siteListStruct
				newSiteForCache.SiteID, _ = strconv.Atoi(JSONSites.Row[index].ID)
				newSiteForCache.SiteName = JSONSites.Row[index].Name
				name := []siteListStruct{newSiteForCache}
				Sites = append(Sites, name...)
				loopCount++
			}
		} else {
			var JSONSites siteRowSingle
			err = json.Unmarshal([]byte(JSONResp.Params.Sites), &JSONSites)
			if err != nil {
				logger(4, "Unable to Unmarshal Site "+err.Error(), false, true)
				return
			}
			var newSiteForCache siteListStruct
			newSiteForCache.SiteID, _ = strconv.Atoi(JSONSites.Row.ID)
			newSiteForCache.SiteName = JSONSites.Row.Name
			name := []siteListStruct{newSiteForCache}
			Sites = append(Sites, name...)
			loopCount++
		}
		bar.Add(loopCount)
		rowStart += loopCount
	}
	bar.FinishPrint("Sites Loaded  \n")
	logger(3, "Sites Loaded: "+strconv.Itoa(len(Sites)), false, true)
}

func getSiteID(u map[string]interface{}, buffer *bytes.Buffer) (siteID int, siteName string) {
	siteNameMapping := fmt.Sprintf("%v", importConf.AssetGenericFieldMapping["h_site"])
	siteName = getFieldValue("h_site", siteNameMapping, u, buffer)
	if siteName != "" && siteName != "__clear__" {
		//-- Check if in Cache
		for _, site := range Sites {
			if strings.EqualFold(site.SiteName, siteName) {
				siteID = site.SiteID
				break
			}
		}
	}
	debugLog(buffer, "Site Mapping:", siteNameMapping, ":", siteName, ":", strconv.Itoa(siteID))
	return
}
