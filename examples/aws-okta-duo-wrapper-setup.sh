#!/bin/bash

echo -n "okta username: "
read -r username

echo -n "okta password: "
read -s password

security delete-generic-password -a $USER -s aws-okta-duo-wrapper-okta-username > /dev/null 2>&1
security -i <<< "add-generic-password -a $USER -s aws-okta-duo-wrapper-okta-username -w '$username'"

security delete-generic-password -a $USER -s aws-okta-duo-wrapper-okta-password > /dev/null 2>&1
security -i <<< "add-generic-password -a $USER -s aws-okta-duo-wrapper-okta-password -w '$password'"
