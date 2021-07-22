package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"

	apiLib "github.com/hornbill/goApiLib"
	"github.com/jmoiron/sqlx"
)

func getSoftwareRecords(u map[string]interface{}, assetType assetTypesStruct, espXmlmc *apiLib.XmlmcInstStruct, db *sqlx.DB, buffer *bytes.Buffer) (softwareRecords map[string]map[string]interface{}, softwareRecordsHash string, err error) {
	if configCSV {
		return
	}

	if assetType.SoftwareInventory.Query != "" && assetType.SoftwareInventory.AssetIDColumn != "" {
		if val, ok := u[assetType.SoftwareInventory.AssetIDColumn]; ok {
			swAssetID := iToS(val)
			debugLog(buffer, "Asset ID found in DB record:", swAssetID)
			softwareRecords, softwareRecordsHash, err = querySoftwareInventoryRecords(swAssetID, assetType, db, buffer)
			if err != nil {
				err = errors.New("Unable to read software inventory records from source DB:" + err.Error())
			}
		} else {
			err = errors.New("unable to read software inventory records from source db, asset ID not found in db record")
		}
	}
	return
}

func buildSoftwareInventory(softwareRecords map[string]map[string]interface{}, assetType assetTypesStruct, hbAssetID string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) {
	countSuccess := 0
	buffer.WriteString(loggerGen(1, strconv.Itoa(len(softwareRecords))+" Software Inventory Records processing..."))
	for k, v := range softwareRecords {
		_, err := addSoftwareInventoryRecord(hbAssetID, v, assetType, espXmlmc, buffer)
		if err != nil {
			buffer.WriteString(loggerGen(4, "Error creating software record ["+k+"]:"+err.Error()))
			mutexCounters.Lock()
			counters.softwareCreateFailed++
			mutexCounters.Unlock()
		} else {
			countSuccess++
		}
	}
	buffer.WriteString(loggerGen(1, strconv.Itoa(countSuccess)+" of "+strconv.Itoa(len(softwareRecords))+" added successfully"))
}

func addSoftwareInventoryRecord(fkAssetID string, softwareRecord map[string]interface{}, assetType assetTypesStruct, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) (pkid int, err error) {
	espXmlmc.SetParam("application", "com.hornbill.servicemanager")
	espXmlmc.SetParam("entity", "AssetsInstalledSoftware")
	espXmlmc.SetParam("returnModifiedData", "true")
	espXmlmc.OpenElement("primaryEntityData")
	espXmlmc.OpenElement("record")
	espXmlmc.SetParam("h_fk_asset_id", fkAssetID)
	//Get software field mapping
	var packageName string
	for k, v := range assetType.SoftwareInventory.Mapping {
		strMapping := fmt.Sprintf("%v", v)
		value := getFieldValue(k, strMapping, softwareRecord, buffer)
		debugLog(buffer, k, ":", strMapping, ":", value)

		if value != "" {
			espXmlmc.SetParam(k, value)
			if k == "h_app_name" {
				packageName = value
			}
		} else {
			if k == "h_app_vendor" || k == "h_app_version" {
				espXmlmc.SetParam(k, "No Value")
			}
		}
	}
	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("primaryEntityData")
	XMLSTRING := espXmlmc.GetParam()
	debugLog(buffer, "Software Record Create XML:", XMLSTRING)
	XMLUpdate, xmlmcErr := espXmlmc.Invoke("data", "entityAddRecord")
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
		err = errors.New("API Call failed when creating software inventory record:" + xmlmcErr.Error())
		return
	}

	var xmlRespon xmlmcResponse
	err = xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
		err = errors.New("Unable to read response from Hornbill instance when creating software inventory record:" + err.Error())
		return
	}

	if xmlRespon.MethodResult != "ok" {
		buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
		err = errors.New("Unable to create software inventory record: " + xmlRespon.State.ErrorRet)
		return
	}
	pkid = xmlRespon.Params.HPKID
	debugLog(buffer, "Software inventory record successfully created: "+strconv.Itoa(pkid)+" - "+packageName)
	mutexCounters.Lock()
	counters.softwareCreated++
	mutexCounters.Unlock()
	return
}

func deleteSoftwareInventoryRecord(pkid int, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) (err error) {
	espXmlmc.SetParam("application", "com.hornbill.servicemanager")
	espXmlmc.SetParam("entity", "AssetsInstalledSoftware")
	espXmlmc.SetParam("keyValue", strconv.Itoa(pkid))
	XMLSTRING := espXmlmc.GetParam()
	debugLog(buffer, "Software Record Delete XML:", XMLSTRING)
	XMLUpdate, xmlmcErr := espXmlmc.Invoke("data", "entityDeleteRecord")
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
		err = errors.New("API Call failed when deleting software inventory record:" + xmlmcErr.Error())
		return
	}

	var xmlRespon xmlmcResponse
	err = xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
		err = errors.New("Unable to read response from Hornbill instance when deleting software inventory record:" + err.Error())
		return
	}

	if xmlRespon.MethodResult != "ok" {
		buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
		err = errors.New("Unable to delete software inventory record: " + xmlRespon.State.ErrorRet)
		return
	}
	debugLog(buffer, "Software inventory record successfully deleted: "+strconv.Itoa(pkid))
	mutexCounters.Lock()
	counters.softwareRemoved++
	mutexCounters.Unlock()
	return
}

func getAssetSoftwareCount(assetID string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) (softwareCount uint64, err error) {
	espXmlmc.SetParam("application", "com.hornbill.servicemanager")
	espXmlmc.SetParam("queryName", "Asset.getInstalledSoftware")
	espXmlmc.OpenElement("queryParams")
	espXmlmc.SetParam("assetId", assetID)
	espXmlmc.CloseElement("queryParams")
	espXmlmc.OpenElement("queryOptions")
	espXmlmc.SetParam("resultType", "count")
	espXmlmc.CloseElement("queryOptions")

	XMLSTRING := espXmlmc.GetParam()
	debugLog(buffer, "Software Record Get XML:", XMLSTRING)

	RespBody, err := espXmlmc.Invoke("data", "queryExec")
	var XMLResp xmlmcSoftwareRecordsResponse
	if err != nil {
		return
	}
	err = xml.Unmarshal([]byte(RespBody), &XMLResp)
	if err != nil {
		return
	}
	if XMLResp.State.Error != "" {
		err = errors.New(XMLResp.State.Error)
		return
	}

	//-- return Count
	softwareCount = XMLResp.Params.RowData.Row[0].Count
	return
}

func getAssetSoftwareRecords(assetID string, assetCount uint64, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) (map[string]softwareRecordDetailsStruct, error) {
	var (
		loopCount uint64
		recordMap = make(map[string]softwareRecordDetailsStruct)
		err       error
	)
	assetPageSize := 1000

	//-- Init Map
	//-- Load Results in pages of assetPageSize
	RespBody := ""
	for loopCount < assetCount {
		espXmlmc.SetParam("application", "com.hornbill.servicemanager")
		espXmlmc.SetParam("queryName", "Asset.getInstalledSoftware")
		espXmlmc.OpenElement("queryParams")
		espXmlmc.SetParam("assetId", assetID)
		espXmlmc.SetParam("rowstart", strconv.FormatUint(loopCount, 10))
		espXmlmc.SetParam("limit", strconv.Itoa(assetPageSize))
		espXmlmc.CloseElement("queryParams")
		espXmlmc.OpenElement("queryOptions")
		espXmlmc.SetParam("resultType", "data")
		espXmlmc.CloseElement("queryOptions")

		XMLSTRING := espXmlmc.GetParam()
		debugLog(buffer, "Software Record Get XML:", XMLSTRING)

		RespBody, err = espXmlmc.Invoke("data", "queryExec")
		var JSONResp xmlmcSoftwareRecordsResponse
		if err != nil {
			err = errors.New("Error returning page of asset records: " + err.Error())
			break
		}
		err = xml.Unmarshal([]byte(RespBody), &JSONResp)
		if err != nil {
			err = errors.New("Error returning page of asset records: " + err.Error())
			break
		}
		if JSONResp.State.Error != "" {
			err = errors.New("Error returning page of asset records: " + JSONResp.State.Error)
			break
		}

		// Add page size
		loopCount += uint64(assetPageSize)

		//-- Check for empty result set
		if len(JSONResp.Params.RowData.Row) == 0 {
			break
		}
		for _, v := range JSONResp.Params.RowData.Row {
			if v.HAppID != "" {
				keyVal := v.HAppID
				recordMap[keyVal] = v
			}
		}
	}
	buffer.WriteString(loggerGen(1, strconv.Itoa(len(recordMap))+" software records cached from Hornbill for Asset ID "+assetID))

	return recordMap, err
}

func updateAssetSI(assetID string, softwareRecords map[string]map[string]interface{}, softwareRecordsHash string, assetType assetTypesStruct, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) (err error) {
	//Get SI records for asset
	//Remove HB SI records that don't exist in DB SI source
	//Add new HB SI records that exist in DB SI source but don't exist in HB SI records against the asset being processed
	var (
		hbSIRecordCount        uint64
		hbSICache              = make(map[string]softwareRecordDetailsStruct)
		boolUpdateSoftwareHash = true
	)
	buffer.WriteString(loggerGen(1, "Processing Software Inventory updates for asset: "+assetID))
	//Process Software Inventory updates
	hbSIRecordCount, err = getAssetSoftwareCount(assetID, espXmlmc, buffer)
	if err != nil {
		buffer.WriteString(loggerGen(4, "Unable to count asset software inventory records: "+err.Error()))
		err = errors.New("Unable to count asset software inventory records: " + err.Error())
	} else {
		hbSICache, err = getAssetSoftwareRecords(assetID, hbSIRecordCount, espXmlmc, buffer)
		if err != nil {
			buffer.WriteString(loggerGen(4, "Unable to cache asset software inventory records: "+err.Error()))
			err = errors.New("Unable to cache asset software inventory records: " + err.Error())
		}
	}
	if len(hbSICache) > 0 {
		//Loop through HB SI cache for this asset, see if match exists in softwareRecords. If not exists, delete
		for cK, cV := range hbSICache {
			//loop through softwareRecords
			delRec := true
			for sK := range softwareRecords {
				if sK == cK {
					delRec = false
				}
			}
			if delRec {
				err = deleteSoftwareInventoryRecord(cV.HPKID, espXmlmc, buffer)
				if err != nil {
					mutexCounters.Lock()
					counters.softwareRemoveFailed++
					mutexCounters.Unlock()
					buffer.WriteString(loggerGen(4, "Error deleting software inventory record: "+err.Error()))
					boolUpdateSoftwareHash = false
				}
			}
		}

		//Loop through softwareRecords, if match doesn't exist in HB SI cache for this asset then add new software record to Hornbill asset
		for sK, sV := range softwareRecords {
			//loop through cache now
			addRec := true
			for cK := range hbSICache {
				if sK == cK {
					addRec = false
				}
			}
			if addRec {
				_, err := addSoftwareInventoryRecord(assetID, sV, assetType, espXmlmc, buffer)
				if err != nil {
					buffer.WriteString(loggerGen(4, "Error creating software record:"+err.Error()))
					mutexCounters.Lock()
					counters.softwareCreateFailed++
					mutexCounters.Unlock()
					boolUpdateSoftwareHash = false
				}
			}
		}

	} else {
		buildSoftwareInventory(softwareRecords, assetType, assetID, espXmlmc, buffer)
	}

	if boolUpdateSoftwareHash {
		//Update software inventory hash
		assetEntity := "AssetsComputer"
		if assetType.Class == "mobileDevice" {
			assetEntity = "AssetsMobileDevice"
		}
		espXmlmc.SetParam("application", "com.hornbill.servicemanager")
		espXmlmc.SetParam("entity", assetEntity)
		espXmlmc.SetParam("returnModifiedData", "false")
		espXmlmc.OpenElement("primaryEntityData")
		espXmlmc.OpenElement("record")
		espXmlmc.SetParam("h_pk_asset_id", assetID)
		espXmlmc.SetParam("h_dsc_sw_fingerprint", softwareRecordsHash)
		espXmlmc.CloseElement("record")
		espXmlmc.CloseElement("primaryEntityData")

		XMLSTRING := espXmlmc.GetParam()
		var XMLUpdate string
		XMLUpdate, err = espXmlmc.Invoke("data", "entityUpdateRecord")

		if err != nil {
			buffer.WriteString(loggerGen(3, "API Call failed when Updating Asset Software Inventory ID:"+err.Error()))
			buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
			return
		}

		var xmlRespon xmlmcResponse
		err = xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
		if err != nil {
			buffer.WriteString(loggerGen(3, "Unable to read response from Hornbill instance when Updating Asset Software Inventory ID:"+err.Error()))
			buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
			return
		}

		if xmlRespon.MethodResult != "ok" {
			buffer.WriteString(loggerGen(3, "Unable to update Asset Software Inventory ID: "+xmlRespon.State.ErrorRet))
			buffer.WriteString(loggerGen(1, "API Call XML: "+XMLSTRING))
			return
		}
	}

	return
}
