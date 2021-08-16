package main

import (
	"bytes"
	"encoding/xml"
	_ "fmt"
	"strings"

	apiLib "github.com/hornbill/goApiLib"
)

func supplierManagerInstalled() bool {
	return isAppInstalled("com.hornbill.suppliermanager")
}
func isAppInstalled(app string) bool {
	for _, b := range HInstalledApplications {
		if strings.EqualFold(b, app) {
			return true
		}
	}
	return false
}

type xmlmcApplicationResponse struct {
	MethodResult string      `xml:"status,attr"`
	Apps         []appStruct `xml:"params>application"`
	State        stateStruct `xml:"state"`
}
type appStruct struct {
	Application string `xml:"name"`
}

func getApplications() {
	espXmlmc := apiLib.NewXmlmcInstance(SQLImportConf.InstanceID)
	espXmlmc.SetAPIKey(SQLImportConf.APIKey)

	XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("session", "getApplicationList")
	if xmlmcErr != nil {
		logger(4, "API Call failed when trying to get application list:"+xmlmcErr.Error(), true, true)
		return
	}
	var xmlRespon xmlmcApplicationResponse
	err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
	if err != nil {
		logger(3, "Failed to read applications: "+err.Error(), true, true)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(3, "Failed to deal with applications: "+xmlRespon.State.ErrorRet, true, true)
		} else {
			var l = len(xmlRespon.Apps)
			for i := 0; i < l; i++ {
				HInstalledApplications = append(HInstalledApplications, xmlRespon.Apps[i].Application)
			}
		}
	}

}

// connectSupplier -- Connect a Supplier to an asset
func connectSupplier(assetId string, supplierId string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) bool {
	boolReturn := false
	//--
	espXmlmc.SetParam("supplierId", supplierId)
	espXmlmc.SetParam("assetId", assetId)

	var XMLSTRING = espXmlmc.GetParam()

	if !configDryRun {

		XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("apps/com.hornbill.suppliermanager/SupplierAssets", "addSupplierAsset")
		if xmlmcErr != nil {
			buffer.WriteString(loggerGen(4, "API Call failed when matching Asset to Supplier:"+xmlmcErr.Error()))
			buffer.WriteString(loggerGen(1, "API XML: "+XMLSTRING))
			return boolReturn
		}
		var xmlRespon xmlmcResponse

		err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
		if err != nil {
			buffer.WriteString(loggerGen(4, "Failed to connect supplier to asset:"+err.Error()))
			buffer.WriteString(loggerGen(1, "API XML: "+XMLSTRING))
			return boolReturn
		} else {
			if xmlRespon.MethodResult != "ok" {
				buffer.WriteString(loggerGen(4, "Failed to connect supplier to asset:"+xmlRespon.State.ErrorRet))
				buffer.WriteString(loggerGen(1, "API XML: "+XMLSTRING))
				return boolReturn
			} else {
				boolReturn = true
			}
		}
	} else {
		buffer.WriteString(loggerGen(1, "Asset Supplier XML "+XMLSTRING))
		espXmlmc.ClearParam()
		boolReturn = true
	}
	return boolReturn
}

// addContract -- Associate a contract to an asset
func addContract(assetId string, contractId string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) bool {
	boolReturn := false
	//--
	espXmlmc.SetParam("supplierContractId", contractId)
	espXmlmc.SetParam("assetId", assetId)

	var XMLSTRING = espXmlmc.GetParam()
	if !configDryRun {
		XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("apps/com.hornbill.suppliermanager/SupplierContractAssets", "addSupplierContractAsset")
		if xmlmcErr != nil {
			buffer.WriteString(loggerGen(4, "API Call failed when matching Asset to Contract: "+xmlmcErr.Error()))
			buffer.WriteString(loggerGen(1, "API XML: "+XMLSTRING))
			return boolReturn
		}
		var xmlRespon xmlmcResponse

		err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
		if err != nil {
			buffer.WriteString(loggerGen(4, "Failed to connect asset to contract: "+err.Error()))
			buffer.WriteString(loggerGen(1, "API XML: "+XMLSTRING))
			return boolReturn
		} else {
			if xmlRespon.MethodResult != "ok" {
				buffer.WriteString(loggerGen(4, "Failed to connect asset to contract: "+xmlRespon.State.ErrorRet))
				buffer.WriteString(loggerGen(1, "API XML: "+XMLSTRING))
				return boolReturn
			} else {
				boolReturn = true
			}
		}
	} else {
		buffer.WriteString(loggerGen(1, "Asset Contract XML "+XMLSTRING))
		espXmlmc.ClearParam()
		boolReturn = true
	}
	return boolReturn
}

// addInPolicy --
func addInPolicy(assetId string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) bool {
	boolReturn := false

	espXmlmc.SetParam("application", "com.hornbill.configurationmanager")
	espXmlmc.SetParam("entity", "ConfigurationItemsInPolicy")
	espXmlmc.SetParam("returnModifiedData", "false")
	espXmlmc.OpenElement("primaryEntityData")
	espXmlmc.OpenElement("record")
	espXmlmc.SetParam("h_entity_id", assetId)
	espXmlmc.SetParam("h_entity_name", "asset")
	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("primaryEntityData")

	var XMLSTRING = espXmlmc.GetParam()
	if !configDryRun {
		XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("data", "entityAddRecord")
		if xmlmcErr != nil {
			buffer.WriteString(loggerGen(4, "API Call failed when trying to bring asset in policy: "+xmlmcErr.Error()))
			buffer.WriteString(loggerGen(1, "API XML: "+XMLSTRING))
			return boolReturn
		}
		var xmlRespon xmlmcResponse

		err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
		if err != nil {
			buffer.WriteString(loggerGen(4, "Failed to bring asset in policy: "+err.Error()))
			buffer.WriteString(loggerGen(1, "API XML: "+XMLSTRING))
			return boolReturn
		} else {
			if xmlRespon.MethodResult != "ok" {
				buffer.WriteString(loggerGen(4, "Failed to bring asset in policy: "+xmlRespon.State.ErrorRet))
				buffer.WriteString(loggerGen(1, "API XML: "+XMLSTRING))
				return boolReturn
			} else {
				boolReturn = true
			}
		}
	} else {
		buffer.WriteString(loggerGen(1, "Asset Policy XML: "+XMLSTRING))
		espXmlmc.ClearParam()
		boolReturn = true
	}
	return boolReturn
}
