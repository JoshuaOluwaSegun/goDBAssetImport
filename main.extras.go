package main

import (
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
