package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hornbill/pb"
)

func loadUsers() {
	//-- Init One connection to Hornbill to load all data
	initXMLMC()
	logger(1, "Loading Users from Hornbill", false, true)

	count := getCount("getUserAccountsList")
	logger(1, "getUserAccountsList Count: "+strconv.FormatUint(count, 10), false, true)
	getUserAccountList(count)

	logger(1, "Users Loaded: "+strconv.Itoa(len(Customers)), false, true)
}

func getUserAccountList(count uint64) {
	var loopCount uint64
	pageSize = 1000
	//-- Init Map
	//-- Load Results in pages of pageSize
	bar := pb.StartNew(int(count))
	for loopCount < count {
		logger(1, "Loading User Accounts List Offset: "+fmt.Sprintf("%d", loopCount)+"\n", false, true)

		hornbillImport.SetParam("application", "com.hornbill.core")
		hornbillImport.SetParam("queryName", "getUserAccountsList")
		hornbillImport.OpenElement("queryParams")
		hornbillImport.SetParam("rowstart", strconv.FormatUint(loopCount, 10))
		hornbillImport.SetParam("limit", strconv.Itoa(pageSize))
		hornbillImport.CloseElement("queryParams")
		RespBody, xmlmcErr := hornbillImport.Invoke("data", "queryExec")

		var JSONResp xmlmcUserListResponse
		if xmlmcErr != nil {
			logger(4, "Unable to Query Accounts List "+xmlmcErr.Error(), false, true)
			break
		}
		err := json.Unmarshal([]byte(RespBody), &JSONResp)
		if err != nil {
			logger(4, "Unable to Query Accounts List "+err.Error(), false, true)
			break
		}
		if JSONResp.State.Error != "" {
			logger(4, "Unable to Query Accounts List "+JSONResp.State.Error, false, true)
			break
		}
		//-- Push into Map
		for index := range JSONResp.Params.RowData.Row {
			var newCustomerForCache customerListStruct
			switch SQLImportConf.HornbillUserIDColumn {
			case "h_employee_id":
				{
					newCustomerForCache.CustomerID = JSONResp.Params.RowData.Row[index].HEmployeeID
				}
			case "h_login_id":
				{
					newCustomerForCache.CustomerID = JSONResp.Params.RowData.Row[index].HLoginID
				}
			case "h_email":
				{
					newCustomerForCache.CustomerID = JSONResp.Params.RowData.Row[index].HEmail
				}
			case "h_name":
				{
					newCustomerForCache.CustomerID = JSONResp.Params.RowData.Row[index].HName
				}
			case "h_attrib_1":
				{
					newCustomerForCache.CustomerID = JSONResp.Params.RowData.Row[index].HAttrib1
				}
			case "h_user_id":
				{
					newCustomerForCache.CustomerID = JSONResp.Params.RowData.Row[index].HUserID
				}
			default:
				{
					newCustomerForCache.CustomerID = JSONResp.Params.RowData.Row[index].HUserID
				}
			}
			newCustomerForCache.CustomerHandle = JSONResp.Params.RowData.Row[index].HUserID
			newCustomerForCache.CustomerName = JSONResp.Params.RowData.Row[index].HFirstName + " " + JSONResp.Params.RowData.Row[index].HLastName
			customerNamedMap := []customerListStruct{newCustomerForCache}
			mutexCustomers.Lock()
			Customers = append(Customers, customerNamedMap...)
			mutexCustomers.Unlock()
		}

		// Add 100
		loopCount += uint64(pageSize)
		bar.Add(len(JSONResp.Params.RowData.Row))
		//-- Check for empty result set
		if len(JSONResp.Params.RowData.Row) == 0 {
			break
		}
	}
	bar.FinishPrint("Accounts Loaded  \n")
}

func getCount(query string) uint64 {

	hornbillImport.SetParam("application", "com.hornbill.core")
	hornbillImport.SetParam("queryName", query)
	hornbillImport.OpenElement("queryParams")
	hornbillImport.SetParam("getCount", "true")
	hornbillImport.CloseElement("queryParams")

	RespBody, xmlmcErr := hornbillImport.Invoke("data", "queryExec")

	var JSONResp xmlmcCountResponse
	if xmlmcErr != nil {
		logger(4, "Unable to run Query ["+query+"] "+xmlmcErr.Error(), false, true)
		return 0
	}
	err := json.Unmarshal([]byte(RespBody), &JSONResp)
	if err != nil {
		logger(4, "Unable to run Query ["+query+"] "+err.Error(), false, true)
		return 0
	}
	if JSONResp.State.Error != "" {
		logger(4, "Unable to run Query ["+query+"] "+JSONResp.State.Error, false, true)
		return 0
	}

	//-- return Count
	count, errC := strconv.ParseUint(JSONResp.Params.RowData.Row[0].Count, 10, 16)
	//-- Check for Error
	if errC != nil {
		logger(4, "Unable to get Count for Query ["+query+"] "+err.Error(), false, true)
		return 0
	}
	return count
}

func getUserID(u map[string]interface{}, userCol string, buffer *bytes.Buffer) (userID, userURN, userName string) {
	userMapping := fmt.Sprintf("%v", SQLImportConf.AssetGenericFieldMapping[userCol])
	userID = getFieldValue(userCol, userMapping, u, buffer)
	if userID != "" && userID != "<nil>" && userID != "__clear__" {
		mutexCustomers.Lock()
		for _, customer := range Customers {
			if strings.EqualFold(customer.CustomerID, userID) {
				userName = customer.CustomerName
				break
			}
		}
		mutexCustomers.Unlock()
	}
	if userName != "" {
		userURN = "urn:sys:0:" + userName + ":" + userID
	}
	debugLog(buffer, "User Mapping:", userCol, ":", userMapping, ":", userID, ":", userName, ":", userURN)
	return
}
