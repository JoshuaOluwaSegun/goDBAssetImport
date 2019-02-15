# CHANGELOG

## 1.7.3 (February 15th, 2019)

Changes:

- Extra gating to prevent duplicate records being created when there's a failure of the API call that checks if the asset already exists on the Hornbill instance

## 1.7.2 (January 4th, 2019)

Changes:

- Record data not correctly mapped when using certain ODBC drivers
- Code tweaks for minor performance improvements

## 1.7.1 (December 28th, 2018)

Defect fix:

- Last Logged On User to URN conversion when updating existing assets
  
## 1.7.0 (December 13th, 2018)

Features:

- Added support to use ODBC SQL Server driver as a data source

## 1.6.0 (December 10th, 2018)

Features:

- Added support for populating the company fields against an asset. The tool will perform a Company look-up if a company name (in the h_company__name mapping) has been provided, before populating the company name and ID fields against the new or updated asset
- Additional logging
- Removed need to provide zone CLI parameter

## 1.5.0 (August 15th, 2018)

Features:

- Added support for searching other entity columns for existing asset records to prevent asset duplication
- Removed mandatory status of username and password columns when authentication method is Windows

## 1.4.2 (April 23rd, 2018)

Feature:

- Added account verification and URN building when value supplied to h_last_logged_on_user column

## 1.4.1 (January 25th, 2018)

Defect fix:

- Fixed issue with Used By not being populated with a valid URN

## 1.4.0 (December 4th, 2017)

Features:

- Adds Asset URN to record once asset record has been created
- Updates Asset URN during asset update

## 1.3.2 (April 3rd, 2017)

Features:

- Added support for Windows authentication against MSSQL Server
- Added example configuration files for all asset types

## 1.3.1 (February 22nd, 2017)

Defect fix:

- Removed unnecessary double-quotes from configuration file

## 1.3.0 (February 1st, 2017)

Features:

- Refactored code in to separate Go files, for easier maintenance
- Provided a more detailed log output when errors occur
- The tool now supports a configuration defined maximum log file size, and will create multiple log files for an import where necessary

Defects fixed:

- Updating Last User or Owner columns for existing assets replaced the user URN with a user ID
- Updating a Primary column required a change to a Related entity columm

## 1.2.1 (October 25th, 2016)

Features:

- Removed specification of Asset Owner or Used By as Hornbill Contacts, to be consistent with the Service Manager application

## 1.2.0 (October 24th, 2016)

Features:

- Replaced Hornbill Instance Username and Password authentication with API Key functionality
- Improved performance by adding support for concurrent import API calls
- Added ability to specify whether the Asset Owner and Asset Used By records are Hornbill Contacts or Hornbill Users

## 1.1.2 (February 15, 2016)

Features:

- Bug fix: NULL values within asset records being replaced by the string `<nil>`

## 1.1.1 (February 03, 2016)

Features:

- Bug fix: Mapping name was being written to asset columns when column value from database was blank or NULL

## 1.1.0 (January 19, 2016)

Features:

- Added support for MySQL versions 3.2.0 to 4.0

## 1.0.0 (December 22, 2015)

Features:

- Initial Release