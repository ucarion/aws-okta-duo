#!/bin/bash

session_id=$(security find-generic-password -s aws-okta-duo-wrapper-okta-session-id -w 2>/dev/null)
username=$(security find-generic-password -s aws-okta-duo-wrapper-okta-username -w)
password=$(security find-generic-password -s aws-okta-duo-wrapper-okta-password -w)

AWS_OKTA_DUO_OKTA_SESSION_ID=$session_id \
  AWS_OKTA_DUO_OKTA_HOST=fill_this_in_for_your_organization \
  AWS_OKTA_DUO_OKTA_USERNAME=$username \
  AWS_OKTA_DUO_OKTA_PASSWORD=$password \
  AWS_OKTA_DUO_OKTA_APP_PATH=fill_this_in_for_your_organization \
  AWS_OKTA_DUO_SAVE_SESSION_CMD=./examples/aws-okta-duo-wrapper-save-session.sh \
  aws-okta-duo $@
