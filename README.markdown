Bookmarking Service
==========================

A personal, open source, MVP Delicious / Pinboard.in clone. (Yay microservices!)

Mean to be run in Google App Engine.

Right now very minimal, but more features coming.

Setup
---------------

    $ cp env.sample .env
    $ $EDITOR .env
    $ dev-scripts/start_app.sh
    
Then visit `localhost:8000/admin`, go into the ApiKey data store and change the default API Key.

API Endpoints
---------------

### Adding a bookmark to the repository

		curl --header "Content-Type: application/json" --header "apikey: CHANGE ME" --URL http://localhost:8080/bookmarks --silent --request POST --data "{\"name\": \"hello\", \"url\": \"http://www.wilcoxd.com\", \"tags\": [\"intro\"]}"

### Retrieving bookmarks from repository

		curl --header "Content-Type: application/json" --URL http://localhost:8080/bookmarks --silent

