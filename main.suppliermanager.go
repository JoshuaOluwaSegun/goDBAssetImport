package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"strconv"

	apiLib "github.com/hornbill/goApiLib"
)

func addSupplierToAsset(assetID, supplierID string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) (err error) {
	espXmlmc.SetParam("supplierId", supplierID)
	espXmlmc.SetParam("assetId", assetID)
	XMLSTRING := espXmlmc.GetParam()
	debugLog(buffer, "Add Supplier to Asset Create XML:", XMLSTRING)
	if !configDryRun {

		XMLUpdate, xmlmcErr := espXmlmc.Invoke("apps/com.hornbill.suppliermanager/SupplierAssets", "addSupplierAsset")
		if xmlmcErr != nil {
			buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
			err = errors.New("API Call failed when creating Supplier to Asset relationship record:" + xmlmcErr.Error())
			return
		}

		var xmlRespon xmlmcResponse
		err = xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
		if err != nil {
			buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
			err = errors.New("Unable to read response from Hornbill instance when creating Supplier to Asset relationship record:" + err.Error())
			return
		}

		if xmlRespon.MethodResult != "ok" {
			buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
			err = errors.New("Unable to create Supplier to Asset relationship record: " + xmlRespon.State.ErrorRet)
			return
		}
		if xmlRespon.Params.Outcome != "success" {
			if xmlRespon.Params.Outcome == "failure - the specified supplier asset already exists" {
				counters.suppliersAssociatedSkipped++
				buffer.WriteString(loggerGen(1, "Supplier Asset relationship already exists"))
				return
			}
			err = errors.New("Unable to create Supplier to Asset relationship record - unexpected outcome from SupplierAssets:addSupplierAsset: " + xmlRespon.Params.Outcome)
			return
		}
		debugLog(buffer, "Supplier to Asset relationship record successfully created: "+strconv.Itoa(xmlRespon.Params.SupplierAssetID))
	}
	mutexCounters.Lock()
	counters.suppliersAssociatedSuccess++
	mutexCounters.Unlock()
	return
}

func addSupplierContractToAsset(assetID, contractID string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) (err error) {
	espXmlmc.SetParam("supplierContractId", contractID)
	espXmlmc.SetParam("assetId", assetID)
	XMLSTRING := espXmlmc.GetParam()
	debugLog(buffer, "Add Supplier Contract to Asset Create XML:", XMLSTRING)

	if !configDryRun {

		XMLUpdate, xmlmcErr := espXmlmc.Invoke("apps/com.hornbill.suppliermanager/SupplierContractAssets", "addSupplierContractAsset")
		if xmlmcErr != nil {
			buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
			err = errors.New("API Call failed when creating Supplier Contract to Asset relationship record:" + xmlmcErr.Error())
			return
		}

		var xmlRespon xmlmcResponse
		err = xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
		if err != nil {
			buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
			err = errors.New("Unable to read response from Hornbill instance when creating Supplier Contract to Asset relationship record:" + err.Error())
			return
		}

		if xmlRespon.MethodResult != "ok" {
			buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
			err = errors.New("Unable to create Supplier Contract to Asset relationship record: " + xmlRespon.State.ErrorRet)
			return
		}
		if xmlRespon.Params.Outcome != "success" {
			if xmlRespon.Params.Outcome == "failure - the specified supplier contract asset already exists" {
				counters.supplierContractsAssociatedSkipped++
				buffer.WriteString(loggerGen(1, "Supplier Asset Contract relationship already exists"))
				return
			}
			err = errors.New("Unable to create Supplier Contract to Asset relationship record - unexpected outcome from SupplierContractAssets:addSupplierContractAsset: " + xmlRespon.Params.Outcome)
			return
		}
		debugLog(buffer, "Supplier Contract to Asset relationship record successfully created: "+strconv.Itoa(xmlRespon.Params.SupplierContractAssetID))
	}
	mutexCounters.Lock()
	counters.supplierContractsAssociatedSuccess++
	mutexCounters.Unlock()
	return
}
