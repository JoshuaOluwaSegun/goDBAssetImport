package main

import (
	"encoding/xml"
	"fmt"
	"strconv"

	"github.com/hornbill/goApiLib"
)

// groupInCache -- Function to check if passed-thorugh site name has been cached
// if so, pass back the Site ID
func groupInCache(groupName string, groupType int) (bool, string) {
	boolReturn := false
	groupID := ""
	mutexSite.Lock()
	//-- Check if in Cache
	for _, group := range Groups {
		if group.GroupName == groupName && group.GroupType == groupType {
			boolReturn = true
			groupID = group.GroupID
		}
	}
	mutexSite.Unlock()
	return boolReturn, groupID
}

// seachSite -- Function to check if passed-through  site  name is on the instance
func searchGroup(groupName string, groupType int, espXmlmc *apiLib.XmlmcInstStruct) (bool, string) {
	boolReturn := false
	groupID := ""
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
		logger(1, "API XML: "+fmt.Sprintf("%s", XMLSTRING), false)
		return boolReturn, groupID
	}
	var xmlRespon xmlmcGroupListResponse

	err := xml.Unmarshal([]byte(XMLGroupSearch), &xmlRespon)
	if err != nil {
		logger(3, "Unable to Search for Group: "+fmt.Sprintf("%v", err), true)
		logger(1, "API XML: "+fmt.Sprintf("%s", XMLSTRING), false)
		return boolReturn, groupID
	}
	if xmlRespon.MethodResult != "ok" {
		logger(3, "Unable to Search for Group: "+xmlRespon.State.ErrorRet, true)
		logger(1, "API XML: "+fmt.Sprintf("%s", XMLSTRING), false)
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
