# CHANGELOG

## 3.4.1 (March 31st, 2023)

Change:

- Added example configuration files for new asset classes: System, Data Processing Record.

## 3.4.0 (February 22nd, 2023)

Change:

- Complied with latest Go binaries because of security advisory.

## 3.3.4 (February 20th, 2023)

Fixed:

- Site Count now returning as integer instead of string

## 3.3.3 (October 31st, 2022)

Fixed:

- dates were not being cleared when using `__clear__` or "date_conversion_clear"

## 3.3.2 (October 11th, 2022)

Fixed:

- h_last_logged_on not being populated when a UTC ms format was provided

## 3.3.1 (October 11th, 2022)

Fixed:

- h_last_logged_on_user not able to be mapped into target records

##�3.3.0 (September 23rd, 2022)

Features:

- Added support to import asset records from vmware Workspace One Unified Endpoint Manager 

## 3.2.0 (September 15th, 2022) 

Features:

- Added support to import asset records from Certero
- Added Go template support to the following fields, but also kept backwards compatibility with config files between v3.0.0 & 3.1.2 inclusive:
  - AssetTypes > AssetIdentifier > SourceColumn
  - AssetTypes > SoftwareInventory > AssetIDColumn
  - AssetTypes > SoftwareInventory > AppIDColumn

## 3.1.2 (July 22nd, 2022)

Fixes:

- Fixed h_attrib_8 and h_attrib_1 for the HornbillUserIDColumn, please use "h_attrib8" and "h_attrib1" instead (respectively).

## 3.1.1 (July 13th, 2022)

Features:

- Added h_attrib_8 as one of the options for the HornbillUserIDColumn.

## 3.1.0 (July 3rd, 2022)

Features:

- Added support to import Chrome OS devices from Google Workspace Enterprise

## 3.0.2 (June 20th, 2022)

Features:

- Added '''InPolicy''' key per Asset type. The options are ''yes'' and `__clear__` and anything else (including omission). "yes" will ensure the asset will be set In Policy (used on Create, Update and Both), `__clear__` is only used if the determined action is "Update" - please note that the timeline will be irrecoverably removed.
- Date Conversion template filters added: "date_conversion" and "date_conversion_clear". The latter will CLEAR the field if the transformation doesn't work. to be used: {{ .FieldName | date_conversion "02/01/2006 15:04:05" }} - provide the input format based on the following reference time of "Jan 1st 2006 4 minutes and 5 seconds past 3pm." - the example shown will convert the regular UK/European date time format to the format useable in the Hornbill datetime field. Please note that IF your formatting is already in the Hornbill date time format ("2006-01-02 15:04:05"), you don't need to convert anything.

## 3.0.1 (March 19th, 2022)

Fixed:

- The SourceConfig Query was not being used resulting that AssetType Queries are erroring out


## 3.0.0 (March 10th, 2022)

Features:

- Replaced hard-coded credentials in JSON config with Keysafe keys
- Added LDAP as a data source for importing assets
- Added ability to update all matched asset records of a specific class, ignoring type
- Added auto-update of utility for releases with same major version

Fixed:

- Owned By, Used By and Last Updated By URNs not correctly set when HornbillUserIDColumn property is set to something other than h_user_id
- Better handled incorrect instance ID, outputs a message and ends rather than panicking

## 2.4.0 (September 9th, 2021)

Changes:

- Removed SupplierManagerIntegration property from configuration, as this is catered for by DBContractColumn and DBSupplierColumn options in the AssetIdentifier object
- Removed duplicated supplier & contract logic
- Removed partially-implemented, non-functional in-policy code
- Added DBContractColumn and DBSupplierColumn properties to the AssetIdentifier object in all example configuration files

## 2.3.0 (September 8th, 2021)

Feature:

- Added Supplier Manager integration - you can now define Suppliers and Supplier Contracts against imported assets

## 2.2.1 (August 16th, 2021)

Change:

- Added `__sharedasset__` as a placeholder for the h_used_by field, allowing an asset to be set as shared.

Fixed:

- modification to re-introduce [HBAssetType] as a placeholder - now introduced as `__hbassettype__` - for the Asset Type. Note that this is case-sensitive. 

## 2.2.0 (August 5th, 2021)

Change:

- Added configuration validation to ensure v2.x compatible mappings are used

## 2.1.1 (August 4th, 2021)

Feature:

- added epoch and epoch_clear template filters. The first is to transform epoch to datetime (treating 0 as empty) the other treats 0 and empty as __clear__.

## 2.1.0 (August 2nd, 2021)

Feature:

- Added support for importing asset records from Nexthink

Changes:

- Improved nil-value logic when populating templates
- Simplified bool const comparisons

## 2.0.0 (July 22nd, 2021)

Changes:

- Incorporated goCSVAssetImport
- Using golang text templating instead of the square bracket notation. People WILL need to revisit their configuration files. Invariable replacing [sampleColumn] with {{.sampleColumn}}

Features:
- The ability to bypass the asset comparison check and force updates

Fixes:
- Fix to related data update counter

## 1.16.1 (July 6th, 2021)

Change:

- Rebuilt using latest version of goApiLib, to fix possible issue with connections via a proxy

## 1.16.0 (June 17th, 2021)

Features: 
- Added support for importing Software Asset Management records when creating or updating assets
- Improved logging, including: grouping log details per-asset; basic log details added to instance log
- Added version check against Github repo 

Changes:
- Optimised import process, including the local caching of Hornbill asset records for each class & type 
- Refactored code to remove duplicated & unnecessary code
- Rounded time taken output to the nearest second

Fixes:
- Removed a number of possible race conditions when using multiple workers

## 1.15.0 (May 14th, 2021)

Features:
- Departments are now mapped (if h_department_name is set - h_department_id will be set behind the scenes) - this now matches the behaviour of organisations (the h_company_name field)
- Ability to specific the Hornbill User ID column for matching users (HornbillUserIDColumn; options: h_user_id (default), h_employee_id, h_email, h_name, h_attrib_1 & h_login_id) - please note that last logged on, owned by and used by will use the same field - i.e. one can NOT specify which column to match to individually.

Changes:
- Front-loading of groups (organisations & departments) - to prevent search for each asset
- Front-loading of Sites - to prevent search for each asset
- If no Asset ID is specified the record will be skipped - instead of activating a search for the asset (to check whether the asset has already been created)

Fixes:
- Possible fix whereby h_asset_urn was not populated correctly
- Optimisation added such that front-loading is not happening if the associated fields are not actually configured.

## 1.14.0 (May 5th, 2021)

Fix:

- Fixed issue with generating user URNs on Asset Creation

## 1.13.1 (January 15th, 2021)

Fix:

- Fixed issue with generating user URNs

## 1.13.0 (January 8th, 2021)

Feature:

- When populating User type fields (Owner, Used By, Last Logged On User), we now use the matched User ID from Hornbill when building the corresponding URNs rather than the User ID from the source data to ensure the case is correct.

## 1.12.0 (December 15th, 2020)

Feature:

- Added support to allow for asset types being imported to be restricted to Create/Update/Both operations

## 1.11.0 (December 15th, 2020)

Feature:

- Added support to skip updating state/substate/operational state fields for existing assets

## 1.10.1 (April 29th, 2020)

Defect Fix:

- Fixed asset identifier character encoding issue when assets are queried from ODBC

## 1.10.0 (April 23rd, 2020)

Change:

- Added support to skip updating asset user when asset is shared

## 1.9.1 (April 16th, 2020)

Change:

- Updated code to support Core application and platform changes

## 1.9.0 (January 8th, 2020)

Changes:

- Added support for clearing asset column values

Defect Fix:

- Fixed issue where asset update counts were not always accurate

## 1.8.1 (October 4th, 2019)

Changes:

- Improved handling of interface nil values

## 1.8.0 (July 25th, 2019)

Changes:

- Added debug mode to output additional debug data to the log

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
