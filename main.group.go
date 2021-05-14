package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"

	apiLib "github.com/hornbill/goApiLib"
	"github.com/hornbill/pb"
)

// groupInCache -- Function to check if passed-thorugh group name has been cached
// if so, pass back the Group ID
func groupInCache(groupName string, groupType int) (bool, string) {
	boolReturn := false
	groupID := ""
	if groupName == "" {
		return boolReturn, groupID
	}
	mutexSite.Lock()
	//-- Check if in Cache
	for _, group := range Groups {
		if group.GroupName == groupName && group.GroupType == groupType {
			boolReturn = true
			groupID = group.GroupID
			break
		}
	}
	mutexSite.Unlock()
	return boolReturn, groupID
}

// searchGroup -- Function to check if passed-through  group is on the instance
func searchGroup(groupName string, groupType int, espXmlmc *apiLib.XmlmcInstStruct) (bool, string) {
	boolReturn := false
	groupID := ""
	if groupName == "" {
		return boolReturn, groupID
	}
	//-- ESP Query for site
	espXmlmc.SetParam("application", "com.hornbill.core")
	espXmlmc.SetParam("queryName", "GetGroupByName")
	espXmlmc.OpenElement("queryParams")
	espXmlmc.SetParam("h_name", groupName)
	espXmlmc.SetParam("h_type", strconv.Itoa(groupType))
	espXmlmc.CloseElement("queryParams")

	var XMLSTRING = espXmlmc.GetParam()
	XMLGroupSearch, xmlmcErr := espXmlmc.Invoke("data", "queryExec")
	if xmlmcErr != nil {
		logger(4, "API Call failed when Searching Group:"+fmt.Sprintf("%v", xmlmcErr), false)
		logger(1, "API XML: "+XMLSTRING, false)
		return boolReturn, groupID
	}
	var xmlRespon xmlmcGroupListResponse

	err := xml.Unmarshal([]byte(XMLGroupSearch), &xmlRespon)
	if err != nil {
		logger(3, "Unable to Search for Group: "+fmt.Sprintf("%v", err), true)
		logger(1, "API XML: "+XMLSTRING, false)
		return boolReturn, groupID
	}
	if xmlRespon.MethodResult != "ok" {
		logger(3, "Unable to Search for Group: "+xmlRespon.State.ErrorRet, true)
		logger(1, "API XML: "+XMLSTRING, false)
		return boolReturn, groupID
	}
	//-- Check Response
	if xmlRespon.GroupID != "" {
		groupID = xmlRespon.GroupID
		boolReturn = true
		//-- Add Group to Cache
		var newGroupForCache groupListStruct
		newGroupForCache.GroupID = groupID
		newGroupForCache.GroupName = groupName
		newGroupForCache.GroupType = groupType
		name := []groupListStruct{newGroupForCache}
		mutexGroup.Lock()
		Groups = append(Groups, name...)
		mutexGroup.Unlock()
	}

	return boolReturn, groupID
}

func loadGroups(groups []string) {
	//-- Init One connection to Hornbill to load all data
	initXMLMC()
	logger(1, "Loading Groups from Hornbill", false)

	count := getGroupCount(groups)
	logger(1, "getGroupCount Count: "+fmt.Sprintf("%d", count), false)
	getGroupList(groups, count)

	logger(1, "Groups Loaded: "+fmt.Sprintf("%d", len(Groups)), false)
}

func getGroupList(groups []string, count int) {
	var pageCount int
	pageSize := 20 //because it appears pagesize is limited differently here...
	//-- Init Map
	pageCount = 0
	//-- Load Results in pages of pageSize
	bar := pb.StartNew(int(count))
	for (pageCount * pageSize) < count {
		pageCount++
		logger(1, "Loading Group List Offset: "+fmt.Sprintf("%d", pageCount)+"\n", false)

		hornbillImport.SetParam("singleLevelOnly", "false")
		for _, group := range groups {
			hornbillImport.SetParam("type", group)
		}
		hornbillImport.OpenElement("orderBy")
		hornbillImport.SetParam("column", "h_name")
		hornbillImport.SetParam("direction", "ascending")
		hornbillImport.CloseElement("orderBy")
		hornbillImport.OpenElement("pageInfo")
		hornbillImport.SetParam("pageIndex", strconv.Itoa(pageCount))
		hornbillImport.SetParam("pageSize", strconv.Itoa(pageSize))
		hornbillImport.CloseElement("pageInfo")

		RespBody, xmlmcErr := hornbillImport.Invoke("admin", "groupGetList2")

		var JSONResp xmlmcGroupResponse
		if xmlmcErr != nil {
			logger(4, "Unable to Query Group List "+fmt.Sprintf("%s", xmlmcErr), false)
			break
		}
		err := json.Unmarshal([]byte(RespBody), &JSONResp)
		if err != nil {
			logger(4, "Unable to Query Groups List "+fmt.Sprintf("%s", err), false)
			break
		}
		if JSONResp.State.Error != "" {
			logger(4, "Unable to Query Groups List "+JSONResp.State.Error, false)
			break
		}
		//-- Push into Map

		for index := range JSONResp.Params.Group {
			var newGroupForCache groupListStruct
			newGroupForCache.GroupID = JSONResp.Params.Group[index].ID
			newGroupForCache.GroupName = JSONResp.Params.Group[index].Name
			switch JSONResp.Params.Group[index].Type {
			case "company":
				newGroupForCache.GroupType = 5
			case "department":
				newGroupForCache.GroupType = 2
			}
			name := []groupListStruct{newGroupForCache}
			mutexGroup.Lock()
			Groups = append(Groups, name...)
			mutexGroup.Unlock()
		}

		// Add 100
		bar.Add(len(JSONResp.Params.Group))
		//-- Check for empty result set
		if len(JSONResp.Params.Group) == 0 {
			break
		}
	}
	bar.FinishPrint("Groups Loaded  \n")
}

type xmlmcGroupResponse struct {
	Params struct {
		Group []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
			Name string `json:"name"`
		} `json:"group"`
		MaxPages int `json:"maxPages"`
	} `json:"params"`
	State stateJSONStruct `json:"state"`
}

func getGroupCount(groups []string) int {

	hornbillImport.SetParam("singleLevelOnly", "false")
	for _, group := range groups {
		hornbillImport.SetParam("type", group)
	}
	hornbillImport.OpenElement("orderBy")
	hornbillImport.SetParam("column", "h_name")
	hornbillImport.SetParam("direction", "ascending")
	hornbillImport.CloseElement("orderBy")
	hornbillImport.OpenElement("pageInfo")
	hornbillImport.SetParam("pageIndex", "1")
	hornbillImport.SetParam("pageSize", "1")
	hornbillImport.CloseElement("pageInfo")

	RespBody, xmlmcErr := hornbillImport.Invoke("admin", "groupGetList2")

	var JSONResp xmlmcGroupResponse
	if xmlmcErr != nil {
		logger(4, "Unable to get Group List "+fmt.Sprintf("%s", xmlmcErr), false)
		return 0
	}
	err := json.Unmarshal([]byte(RespBody), &JSONResp)
	if err != nil {
		logger(4, "Unable to get Group List "+fmt.Sprintf("%s", err), false)
		return 0
	}
	if JSONResp.State.Error != "" {
		logger(4, "Unable to get Group List "+JSONResp.State.Error, false)
		return 0
	}

	return JSONResp.Params.MaxPages
}
