package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"

	apiLib "github.com/hornbill/goApiLib"
)

type googleResponseStruct struct {
	Params struct {
		Data struct {
			ChromeOSDevices []map[string]interface{} `json:"chromeosdevices"`
			NextPageToken   string                   `json:"nextPageToken"`
		} `json:"data"`
		Error   string `json:"error"`
		Status  int    `json:"status"`
		Success bool   `json:"success"`
		URL     string `json:"url"`
	} `json:"params"`
}

type googlePayloadStruct struct {
	Customer    string `json:"customerId"`
	MaxResults  int    `json:"maxResults"`
	PageToken   string `json:"pageToken"`
	Query       string `json:"query"`
	OrgUnitPath string `json:"orgUnitPath"`
}

func getAssetsFromGoogle(assetType assetTypesStruct) (map[string]map[string]interface{}, error) {
	//Initialise Asset Map
	returnMap := make(map[string]map[string]interface{})
	logger(3, " ", false, false)
	logger(3, "[GOOGLE] Running Google query for "+assetType.AssetType+" assets. Please wait...", true, true)

	var nextPageToken string

	gEspXmlmc := apiLib.NewXmlmcInstance(importConf.InstanceID)
	gEspXmlmc.SetAPIKey(importConf.APIKey)
	for {
		assetsList, err := getDevicesPageGoogle(gEspXmlmc, nextPageToken)
		if err != nil {
			return returnMap, err
		}
		for _, v := range assetsList.Params.Data.ChromeOSDevices {
			assetIdentifier := fmt.Sprintf("%s", v[assetType.AssetIdentifier.SourceColumn])
			returnMap[assetIdentifier] = v
		}
		// Google's API will return a token even when on the last page of data.
		// So break the loop if no token is returned
		if assetsList.Params.Data.NextPageToken == "" {
			break
		}
		nextPageToken = assetsList.Params.Data.NextPageToken
	}
	if len(returnMap) == 0 {
		logger(3, "No "+assetType.AssetType+" asset records returned from Google - check your configuration!", true, true)
	} else {
		logger(2, "Total "+assetType.AssetType+" asset records returned from Google: "+strconv.Itoa(len(returnMap)), true, true)
	}
	return returnMap, nil
}

func getDevicesPageGoogle(gEspXmlmc *apiLib.XmlmcInstStruct, pageToken string) (assetsResponse googleResponseStruct, err error) {
	var payload = googlePayloadStruct{
		Customer:    importConf.SourceConfig.Google.Customer,
		MaxResults:  200,
		PageToken:   pageToken,
		Query:       importConf.SourceConfig.Google.Query,
		OrgUnitPath: importConf.SourceConfig.Google.OrgUnitPath,
	}

	strPayload, err := json.Marshal(payload)
	if err != nil {
		logger(4, "getDevicesPage::marshal:Error parsing request payload:"+err.Error(), true, true)
		return
	}
	gEspXmlmc.SetParam("methodPath", "/Google/Workspace/DataSources.system/List Devices.m")
	gEspXmlmc.SetParam("requestPayload", string(strPayload))
	gEspXmlmc.OpenElement("credential")
	gEspXmlmc.SetParam("id", "googleworkspace")
	gEspXmlmc.SetParam("keyId", strconv.Itoa(importConf.KeysafeKeyID))
	gEspXmlmc.CloseElement("credential")

	requestPayloadXML := gEspXmlmc.GetParam()
	responsePayloadXML, err := gEspXmlmc.Invoke("bpm", "iBridgeInvoke")
	if err != nil {
		logger(4, "getDevicesPage::iBridgeInvoke:invoke:"+err.Error(), true, true)
		logger(4, "Request XML: "+requestPayloadXML, false, true)
		return
	}
	var xmlRespon xmlmcIBridgeResponse
	err = xml.Unmarshal([]byte(strings.Map(printOnly, string(responsePayloadXML))), &xmlRespon)
	if err != nil {
		logger(4, "getDevicesPage::iBridgeInvoke:unmarshal:"+err.Error(), true, true)
		logger(4, "Request XML: "+requestPayloadXML, false, true)
		logger(4, "Response XML: "+responsePayloadXML, false, true)
		return
	}
	if xmlRespon.MethodResult != "ok" {
		logger(4, "getDevicesPage::iBridgeInvoke:methodResult:"+xmlRespon.State.Error, true, true)
		logger(4, "Request XML: "+requestPayloadXML, false, true)
		logger(4, "Response XML: "+responsePayloadXML, false, true)
		err = errors.New(xmlRespon.State.Error)
		return
	}
	if xmlRespon.IBridgeResponseError != "" {
		logger(4, "getDevicesPage::iBridgeInvoke:responseError:"+xmlRespon.IBridgeResponseError, true, true)
		logger(4, "Request XML: "+requestPayloadXML, false, true)
		logger(4, "Response XML: "+responsePayloadXML, false, true)
		err = errors.New(xmlRespon.IBridgeResponseError)
		return
	}

	err = json.Unmarshal([]byte(xmlRespon.IBridgeResponsePayload), &assetsResponse)
	if err != nil {
		logger(4, "getDevicesPage::iBridgeInvoke:jsonUnmarshal:"+err.Error(), true, true)
		logger(4, "JSON: "+xmlRespon.IBridgeResponsePayload, false, true)
	}
	return
}
