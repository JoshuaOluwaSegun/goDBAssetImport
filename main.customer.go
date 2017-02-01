package main

import (
	"encoding/xml"
	"fmt"
	"github.com/hornbill/goApiLib"
)

// customerInCache -- Function to check if passed-thorugh Customer ID has been cached
// if so, pass back the Customer Name
func customerInCache(customerID string) (bool, string) {
	boolReturn := false
	strReturn := ""
	mutexCustomers.Lock()
	//-- Check if in Cache
	for _, customer := range Customers {
		if customer.CustomerID == customerID {
			boolReturn = true
			strReturn = customer.CustomerName
		}
	}
	mutexCustomers.Unlock()
	return boolReturn, strReturn
}

// seachSite -- Function to check if passed-through  site  name is on the instance
func searchCustomer(custID string, espXmlmc *apiLib.XmlmcInstStruct) (bool, string) {
	boolReturn := false
	strReturn := ""
	//Get Analyst Info
	espXmlmc.SetParam("customerId", custID)
	espXmlmc.SetParam("customerType", "0")
	var XMLSTRING = espXmlmc.GetParam()
	XMLCustomerSearch, xmlmcErr := espXmlmc.Invoke("apps/"+appServiceManager, "shrGetCustomerDetails")
	if xmlmcErr != nil {
		logger(4, "Unable to Search for Customer ["+custID+"]: "+fmt.Sprintf("%v", xmlmcErr), true)
		logger(1, "API XML: "+fmt.Sprintf("%s", XMLSTRING), false)
	}

	var xmlRespon xmlmcCustomerListResponse
	err := xml.Unmarshal([]byte(XMLCustomerSearch), &xmlRespon)
	if err != nil {
		logger(4, "Unable to Search for Customer ["+custID+"]: "+fmt.Sprintf("%v", err), false)
		logger(1, "API XML: "+fmt.Sprintf("%s", XMLSTRING), false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			//Customer most likely does not exist
			logger(4, "Unable to Search for Customer ["+custID+"]: "+xmlRespon.State.ErrorRet, false)
			logger(1, "API XML: "+fmt.Sprintf("%s", XMLSTRING), false)
		} else {
			//-- Check Response
			if xmlRespon.CustomerFirstName != "" {
				boolReturn = true
				//-- Add Customer to Cache
				var newCustomerForCache customerListStruct
				newCustomerForCache.CustomerID = custID
				newCustomerForCache.CustomerName = xmlRespon.CustomerFirstName + " " + xmlRespon.CustomerLastName
				strReturn = newCustomerForCache.CustomerName
				customerNamedMap := []customerListStruct{newCustomerForCache}
				mutexCustomers.Lock()
				Customers = append(Customers, customerNamedMap...)
				mutexCustomers.Unlock()
			}
		}
	}
	return boolReturn, strReturn
}
