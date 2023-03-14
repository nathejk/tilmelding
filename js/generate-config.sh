#!/usr/bin/with-contenv sh

CONF="window.envConfig = {\
    AUTH_BASEURL: \"$AUTH_BASEURL\",\
    API_BASEURL: \"$API_BASEURL\",\
    DEBUG: \"$DEBUG\",\
}"

if [ "$VUECONFIG" = "" ]
then
    echo $CONF
else
    echo $CONF > $VUECONFIG
fi

