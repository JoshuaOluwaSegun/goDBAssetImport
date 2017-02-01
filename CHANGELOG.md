## 1.3.0 (February 1st, 2017)

Features:
  - Refactored code in to separate Go files, for easier maintenance
  - Improved performance by adding support for concurrent asset processing
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

  - Bug fix: NULL values within asset records being replaced by the string <nil>


## 1.1.1 (February 03, 2016)

Features:

  - Bug fix: Mapping name was being written to asset columns when column value from database was blank or NULL


## 1.1.0 (January 19, 2016)

Features:

  - Added support for MySQL versions 3.2.0 to 4.0


## 1.0.0 (December 22, 2015)

Features:

  - Initial Release
