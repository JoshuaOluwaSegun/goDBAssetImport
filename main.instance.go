package main

import ()

// getInstanceURL -- Function to build XMLMC End Point
func getInstanceURL() string {
	XmlmcInstanceConfig.url = "https://"
	XmlmcInstanceConfig.url += XmlmcInstanceConfig.zone
	XmlmcInstanceConfig.url += "api.hornbill.com/"
	XmlmcInstanceConfig.url += XmlmcInstanceConfig.instance
	XmlmcInstanceConfig.url += "/xmlmc/"
	return XmlmcInstanceConfig.url
}

// SetInstance sets the Zone and Instance config from the passed-through strZone and instanceID values
func SetInstance(strZone string, instanceID string) {
	//-- Set Zone
	SetZone(strZone)
	//-- Set Instance
	XmlmcInstanceConfig.instance = instanceID
	return
}

// SetZone - sets the Instance Zone to Overide current live zone
func SetZone(zone string) {
	XmlmcInstanceConfig.zone = zone
	return
}
