package main

import (
	"crypto/tls"
	"fmt"
	"strconv"

	"github.com/bwmarrin/go-objectsid"
	"github.com/mavricknz/ldap"
)

func connectLDAP() *ldap.LDAPConnection {

	TLSconfig := &tls.Config{
		ServerName:         key.Host,
		InsecureSkipVerify: importConf.SourceConfig.LDAP.Server.InsecureSkipVerify,
	}
	//-- Based on Connection Type Normal | TLS | SSL
	if importConf.SourceConfig.LDAP.Server.Debug {
		logger(3, "Attempting Connection to LDAP... \nServer: "+key.Host+"\nPort: "+fmt.Sprintf("%d", key.Port)+"\nType: "+importConf.SourceConfig.LDAP.Server.ConnectionType+"\nSkip Verify: "+fmt.Sprintf("%t", importConf.SourceConfig.LDAP.Server.InsecureSkipVerify)+"\nDebug: "+fmt.Sprintf("%t", importConf.SourceConfig.LDAP.Server.Debug), true, true)
	}

	t := importConf.SourceConfig.LDAP.Server.ConnectionType
	switch t {
	case "":
		//-- Normal
		logger(3, "Creating LDAP Connection", false, true)
		l := ldap.NewLDAPConnection(key.Host, key.Port)
		l.Debug = importConf.SourceConfig.LDAP.Server.Debug
		return l
	case "TLS":
		//-- TLS
		logger(3, "Creating LDAP Connection (TLS)", false, true)
		l := ldap.NewLDAPTLSConnection(key.Host, key.Port, TLSconfig)
		l.Debug = importConf.SourceConfig.LDAP.Server.Debug
		return l
	case "SSL":
		//-- SSL
		logger(3, "Creating LDAP Connection (SSL)", false, true)
		l := ldap.NewLDAPSSLConnection(key.Host, key.Port, TLSconfig)
		l.Debug = importConf.SourceConfig.LDAP.Server.Debug
		return l
	}

	return nil
}

// -- Query LDAP
func queryLDAP(assetType assetTypesStruct) (map[string]map[string]interface{}, bool) {

	logger(3, "LDAP DSN: "+assetType.LDAPDSN, true, true)
	logger(3, "LDAP Query For Assets: "+assetType.Query, true, true)
	ldapAssets := make(map[string]map[string]interface{})
	//-- Create LDAP Connection
	l := connectLDAP()
	err := l.Connect()
	if err != nil {
		logger(4, "Connecting Error: "+err.Error(), true, true)
		return ldapAssets, false
	}
	defer l.Close()

	//-- Bind
	err = l.Bind(key.Username, key.Password)
	if err != nil {
		logger(4, "Bind Error: "+err.Error(), true, true)
		return ldapAssets, false
	}
	if importConf.SourceConfig.LDAP.Server.Debug {
		logger(3, "LDAP Search Query \n"+fmt.Sprintf("%+v", importConf.SourceConfig.LDAP.Query)+" ----", false, true)
	}
	//-- Build Search Request
	searchRequest := ldap.NewSearchRequest(
		assetType.LDAPDSN,
		importConf.SourceConfig.LDAP.Query.Scope,
		importConf.SourceConfig.LDAP.Query.DerefAliases,
		importConf.SourceConfig.LDAP.Query.SizeLimit,
		importConf.SourceConfig.LDAP.Query.TimeLimit,
		importConf.SourceConfig.LDAP.Query.TypesOnly,
		assetType.Query,
		importConf.SourceConfig.LDAP.Query.Attributes,
		nil)

	//-- Search Request with 1000 limit pagaing
	results, err := l.SearchWithPaging(searchRequest, 1000)
	if err != nil {
		logger(4, "Search Error: "+err.Error(), true, true)
		return ldapAssets, false
	}

	logger(3, "LDAP Results: "+fmt.Sprintf("%d", len(results.Entries))+"\n", true, true)
	//-- Catch zero results
	if len(results.Entries) == 0 {
		logger(4, "No Assets Found ", true, true)
		return ldapAssets, false
	}

	for _, asset := range results.Entries {
		assetIdentifier := asset.GetAttributeValue(assetType.AssetIdentifier.SourceColumn)
		ldapAssets[assetIdentifier] = make(map[string]interface{})
		for _, v := range importConf.SourceConfig.LDAP.Query.Attributes {
			if v == "objectSid" {
				sid := objectsid.Decode([]byte(asset.GetAttributeValue(v)))
				ldapAssets[assetIdentifier][v] = sid.String()
			} else if v == "objectGUID" {
				ldapAssets[assetIdentifier][v] = convertOctetStringToGuid(asset.GetAttributeValue(v))
			} else {
				ldapAssets[assetIdentifier][v] = asset.GetAttributeValue(v)
			}
		}
	}
	return ldapAssets, true
}

func convertOctetStringToGuid(octetString string) string {
	var returnString string
	var o []int
	octetBytes := []byte(octetString)
	for _, v := range octetBytes {
		octet, _ := strconv.Atoi(fmt.Sprintf("%d", v))
		o = append(o, octet)
	}
	returnString += fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x", o[3], o[2], o[1], o[0], o[5], o[4], o[7], o[6], o[8], o[9], o[10], o[11], o[12], o[13], o[14], o[15])
	return returnString
}
