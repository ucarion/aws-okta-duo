#!/bin/bash

security delete-generic-password -a $USER -s aws-okta-duo-wrapper-okta-session-id > /dev/null 2>&1
security -i <<< "add-generic-password -a $USER -s aws-okta-duo-wrapper-okta-session-id -w '$OKTA_SESSION_ID'"
