# LDAP Proxy

Proxies lookups to KTH's ldap server allowing us to access it from servers not
physically located at KTH and translates it to a nice(?) REST api returning
JSON. Not exposed to the internet.

Also does a few sanity checks and logs warnings when data is not as expected.

## API:

Send a GET request to `/user` including either `kthid` or `ug_kthid` in the query string.

Unless there are any errors, the response will have `Content-Type` set to
`application/json`. Example response body:

```json
{"kthid":"turetek","ug_kthid":"u1jwkms6","first_name":"Ture","surname":"Teknokrat"}
```

If there is an error `Content-Type` will not be set. An error message will be
sent in the body as text. If no user is found the status code will be `404`. If
the request is invalid the status code will be `400`. If something else goes
wrong the sttus code will be `500`.

## Testing locally:

```sh
ssh -L3389:ldap.kth.se:389 mjukglass -N
```

```sh
LISTEN_ADDRESS=:3000 LDAP_URL="ldap://localhost:3389" go run .
```
