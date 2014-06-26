go-image-cache
==============

Simple image caching solution in golang. Provides flexible CORS settings.

# Deployment on Heroku

follow: http://mmcgrana.github.io/2012/09/getting-started-with-go-on-heroku.html


## Configuration

Needs following ENV-variables:

    export MEMCACHED_URL='tcp://user:pass@localhost:11211'
    export ORIGIN=http://origin.server.com
    export PORT=9191

Following ENV-variables are optional:

    export NEW_RELIC_LICENSE_KEY='your new relic key'
    export NEW_RELIC_APP_NAME='your app name'
