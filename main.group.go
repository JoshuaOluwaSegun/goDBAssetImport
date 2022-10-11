package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/cheggaaa/pb"
)

func loadGroups(groups []string) {
	//-- Init One connection to Hornbill to load all data
	logger(3, "Loading Groups from Hornbill", false, true)
	count := getGroupCount(groups)
	logger(3, "getGroupCount Count: "+strconv.Itoa(count), false, true)
	getGroupList(groups, count)
	logger(3, "Groups Loaded: "+strconv.Itoa(len(Groups)), false, true)
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
		logger(4, "Unable to get Group List "+xmlmcErr.Error(), false, true)
		return 0
	}
	err := json.Unmarshal([]byte(RespBody), &JSONResp)
	if err != nil {
		logger(4, "Unable to unmarshal Group List "+err.Error(), false, true)
		return 0
	}
	if JSONResp.State.Error != "" {
		logger(4, "Unable to read Group List "+JSONResp.State.Error, false, true)
		return 0
	}

	return JSONResp.Params.MaxPages
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
		logger(3, "Loading Group List Offset: "+strconv.Itoa(pageCount), false, true)

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
			logger(4, "Unable to get Group List "+xmlmcErr.Error(), false, true)
			break
		}
		err := json.Unmarshal([]byte(RespBody), &JSONResp)
		if err != nil {
			logger(4, "Unable to unmarshal Groups List "+err.Error(), false, true)
			break
		}
		if JSONResp.State.Error != "" {
			logger(4, "Unable to read Groups List "+JSONResp.State.Error, false, true)
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
			Groups = append(Groups, name...)
		}

		bar.Add(len(JSONResp.Params.Group))
		if len(JSONResp.Params.Group) == 0 {
			break
		}
	}
	bar.FinishPrint("Groups Loaded  \n")
}

func getGroupID(u map[string]interface{}, groupType string, buffer *bytes.Buffer) (groupID, groupName string) {
	groupCol := ""
	groupTypeID := 0
	switch groupType {
	case "department":
		groupTypeID = 2
		groupCol = "h_department_name"
	case "company":
		groupTypeID = 5
		groupCol = "h_company_name"
	}
	groupNameMapping := fmt.Sprintf("%v", importConf.AssetGenericFieldMapping[groupCol])
	groupName = getFieldValue(groupCol, groupNameMapping, u, buffer)
	if groupName != "" && groupName != "<nil>" && groupName != "__clear__" {
		//-- Check if group is in Cache
		for _, group := range Groups {
			if strings.EqualFold(group.GroupName, groupName) && group.GroupType == groupTypeID {
				groupID = group.GroupID
				break
			}
		}
	}
	debugLog(buffer, "Group Mapping:", groupCol, ":", groupNameMapping, ":", groupName, ":", groupID)
	return
}
