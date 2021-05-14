package main

import (
	"encoding/xml"
	"fmt"
	"strings"

	apiLib "github.com/hornbill/goApiLib"
)

func supplierManagerInstalled() bool {
	return isAppInstalled("com.hornbill.suppliermanager")
}
func configManagerInstalled() bool {
	return isAppInstalled("com.hornbill.configurationmanager")
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
		logger(4, "API Call failed when trying to bring asset in policy:"+fmt.Sprintf("%v", xmlmcErr), false)
	}
	var xmlRespon xmlmcApplicationResponse
	err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
	if err != nil {
		logger(3, "Failed to read applications: "+fmt.Sprintf("%v", err), true)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(3, "Failed to deal with applications: "+xmlRespon.State.ErrorRet, true)
		} else {
			var l = len(xmlRespon.Apps)
			for i := 0; i < l; i++ {
				HInstalledApplications = append(HInstalledApplications, xmlRespon.Apps[i].Application)
			}
		}
	}

}

// connectSupplier -- Connect a Supplier to an asset
func connectSupplier(assetId string, supplierId string, espXmlmc *apiLib.XmlmcInstStruct) bool {
	boolReturn := false
	//--
	espXmlmc.SetParam("supplierId", supplierId)
	espXmlmc.SetParam("assetId", assetId)

	var XMLSTRING = espXmlmc.GetParam()

	if !configDryRun {

		XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("apps/com.hornbill.suppliermanager/SupplierAssets", "addSupplierAsset")
		if xmlmcErr != nil {
			logger(4, "API Call failed when matching Asset to Supplier:"+fmt.Sprintf("%v", xmlmcErr), false)
			logger(1, "API XML: "+XMLSTRING, false)
		}
		var xmlRespon xmlmcResponse

		err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
		if err != nil {
			logger(3, "Failed to connect supplier to asset: "+fmt.Sprintf("%v", err), true)
			logger(1, "API XML: "+XMLSTRING, false)
		} else {
			if xmlRespon.MethodResult != "ok" {
				logger(3, "Failed to connect supplier to asset: "+xmlRespon.State.ErrorRet, true)
				logger(1, "API XML: "+XMLSTRING, false)
			} else {
				boolReturn = true
			}
		}
	} else {
		logger(1, "Asset Supplier XML "+XMLSTRING, false)
		espXmlmc.ClearParam()
		boolReturn = true
	}
	return boolReturn
}

// addContract -- Associate a contract to an asset
func addContract(assetId string, contractId string, espXmlmc *apiLib.XmlmcInstStruct) bool {
	boolReturn := false
	//--
	espXmlmc.SetParam("supplierContractId", contractId)
	espXmlmc.SetParam("assetId", assetId)

	var XMLSTRING = espXmlmc.GetParam()
	if !configDryRun {
		XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("apps/com.hornbill.suppliermanager/SupplierContractAssets", "addSupplierContractAsset")
		if xmlmcErr != nil {
			logger(4, "API Call failed when matching Asset to Contract:"+fmt.Sprintf("%v", xmlmcErr), false)
			logger(1, "API XML: "+XMLSTRING, false)
		}
		var xmlRespon xmlmcResponse

		err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
		if err != nil {
			logger(3, "Failed to connect asset to contract: "+fmt.Sprintf("%v", err), true)
			logger(1, "API XML: "+XMLSTRING, false)
		} else {
			if xmlRespon.MethodResult != "ok" {
				logger(3, "Failed to connect asset to contract: "+xmlRespon.State.ErrorRet, true)
				logger(1, "API XML: "+XMLSTRING, false)
			} else {
				boolReturn = true
			}
		}
	} else {
		logger(1, "Asset Contract XML "+XMLSTRING, false)
		espXmlmc.ClearParam()
		boolReturn = true
	}
	return boolReturn
}

// addInPolicy --
func addInPolicy(assetId string, espXmlmc *apiLib.XmlmcInstStruct) bool {
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
			logger(4, "API Call failed when trying to bring asset in policy:"+fmt.Sprintf("%v", xmlmcErr), false)
			logger(1, "API XML: "+XMLSTRING, false)
		}
		var xmlRespon xmlmcResponse

		err := xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
		if err != nil {
			logger(3, "Failed to bring asset in policy: "+fmt.Sprintf("%v", err), true)
			logger(1, "API XML: "+XMLSTRING, false)
		} else {
			if xmlRespon.MethodResult != "ok" {
				logger(3, "Failed to bring asset into policy: "+xmlRespon.State.ErrorRet, true)
				logger(1, "API XML: "+XMLSTRING, false)
			} else {
				boolReturn = true
			}
		}
	} else {
		logger(1, "Asset Policy XML "+XMLSTRING, false)
		espXmlmc.ClearParam()
		boolReturn = true
	}
	return boolReturn
}

//get CIs in Policy
//entityBrowseRecords2 - https://beta.hornbill.com/hornbill/workspaces/urn:buzz:activityStream:c044d976-b331-483e-a085-1edc63b57f04/
//add TAG
func getTagList(espXmlmc *apiLib.XmlmcInstStruct) bool {
	//var iPage := 1
	// invoke("library","tagGetList"){
	//}
	return true
} /*
<methodCall service="library" method="tagGetList">
<params>
<tagGroup>urn:tagGroup:serviceManagerAssets</tagGroup>
<nameFilter>blubber</nameFilter>
<pageInfo>
<pageIndex>1</pageIndex>
<pageSize>100</pageSize>
</pageInfo></params>
</methodCall>
{
	"@status": true,
	"params": {
		"language": "en-GB",
		"name": [
			{
				"tagId": 4,
				"text": "blubber"
			}
		],
		"maxPages": 1
	}
}
<methodCall service="library" method="tagCreate">
<params>
<tagGroup>urn:tagGroup:serviceManagerAssets</tagGroup>
<tag><text>thisi is a tag</text>
<language>en-GB</language>
</tag>
</params>
</methodCall>
{
	"@status": true,
	"params": {
		"tagId": "5"
	}
}
<methodCall service="library" method="tagLinkObject">
<params>
<tagGroup>urn:tagGroup:serviceManagerAssets</tagGroup>
<tagId>5</tagId>
<objectRefUrn>urn:sys:entity:com.hornbill.servicemanager:Asset:5487</objectRefUrn>
</params>
</methodCall>
*/
