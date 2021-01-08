package main

import (
	"encoding/xml"
	"fmt"
	"strings"

	apiLib "github.com/hornbill/goApiLib"
)

// customerInCache -- Function to check if passed-thorugh Customer ID has been cached
// if so, pass back the Customer Name
func customerInCache(customerID string) (bool, string, string) {
	boolReturn := false
	customerName := ""
	customerIDReturn := ""
	mutexCustomers.Lock()
	//-- Check if in Cache
	for _, customer := range Customers {
		if strings.EqualFold(customer.CustomerID, customerID) {
			boolReturn = true
			customerName = customer.CustomerName
			customerIDReturn = customer.CustomerID
		}
	}
	mutexCustomers.Unlock()
	return boolReturn, customerName, customerIDReturn
}

// seachSite -- Function to check if passed-through  site  name is on the instance
func searchCustomer(custID string, espXmlmc *apiLib.XmlmcInstStruct) (bool, string, string) {
	boolReturn := false
	custNameReturn := ""
	custIDReturn := ""
	//Get Analyst Info
	espXmlmc.SetParam("customerId", custID)
	espXmlmc.SetParam("customerType", "0")
	var XMLSTRING = espXmlmc.GetParam()
	XMLCustomerSearch, xmlmcErr := espXmlmc.Invoke("apps/"+appServiceManager, "shrGetCustomerDetails")
	if xmlmcErr != nil {
		logger(4, "Unable to Search for Customer ["+custID+"]: "+fmt.Sprintf("%v", xmlmcErr), true)
		logger(1, "API XML: "+XMLSTRING, false)
	}

	var xmlRespon xmlmcCustomerListResponse
	err := xml.Unmarshal([]byte(XMLCustomerSearch), &xmlRespon)
	if err != nil {
		logger(4, "Unable to Search for Customer ["+custID+"]: "+fmt.Sprintf("%v", err), false)
		logger(1, "API XML: "+XMLSTRING, false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			//Customer most likely does not exist
			logger(4, "Unable to Search for Customer ["+custID+"]: "+xmlRespon.State.ErrorRet, false)
			logger(1, "API XML: "+XMLSTRING, false)
		} else {
			//-- Check Response
			if xmlRespon.CustomerFirstName != "" {
				boolReturn = true
				//-- Add Customer to Cache
				var newCustomerForCache customerListStruct
				newCustomerForCache.CustomerID = custID
				newCustomerForCache.CustomerName = xmlRespon.CustomerFirstName + " " + xmlRespon.CustomerLastName
				custNameReturn = newCustomerForCache.CustomerName
				custIDReturn = newCustomerForCache.CustomerID
				customerNamedMap := []customerListStruct{newCustomerForCache}
				mutexCustomers.Lock()
				Customers = append(Customers, customerNamedMap...)
				mutexCustomers.Unlock()
			}
		}
	}
	return boolReturn, custNameReturn, custIDReturn
}
