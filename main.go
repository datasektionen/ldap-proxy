package main

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/go-ldap/ldap/v3"
)

var ldapURL string = os.Getenv("LDAP_URL")
var listenAddress string = os.Getenv("LISTEN_ADDRESS")

func main() {
	http.HandleFunc("/user", lookupUser)

	slog.Info("Server started", "listen address", listenAddress, "ldap url", ldapURL)
	http.ListenAndServe(listenAddress, nil)
}

func lookupUser(w http.ResponseWriter, r *http.Request) {
	// Yes, the `Cancel`-field on `net.Dialer` is deprecated, but `ldap` does
	// not seem to have a way of calling `DialContext`, which is what you
	// should use instead.
	cancel := make(chan struct{})
	go func() {
		select {
		case <-r.Context().Done():
			cancel <- struct{}{}
		}
	}()

	conn, err := ldap.DialURL(ldapURL, ldap.DialWithDialer(&net.Dialer{Cancel: cancel}))
	if err != nil {
		slog.Error("Error from DialURL", "error", err)
		http.Error(w, "Cannot connect to ldap server", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	kthid := r.FormValue("kthid")
	ugKTHID := r.FormValue("ug_kthid")
	if (kthid == "") == (ugKTHID == "") {
		http.Error(w, "Exactly one of `kthid` and `ug_kthid` must be provided", http.StatusBadRequest)
		return
	}

	filter := "(ugUsername=" + ldap.EscapeFilter(kthid) + ")"
	if ugKTHID != "" {
		filter = "(ugKthid=" + ldap.EscapeFilter(ugKTHID) + ")"
	}

	res, err := conn.SearchWithPaging(&ldap.SearchRequest{
		BaseDN:       "ou=Addressbook,dc=kth,dc=se",
		Scope:        ldap.ScopeWholeSubtree,
		DerefAliases: ldap.NeverDerefAliases,
		Filter:       filter,
		Attributes: []string{
			"ugUsername", "ugKthid", "givenName", "sn",
			"displayName", "mail", "cn",
		},
	}, 1)
	if err != nil {
		slog.Error("Error from SearchWithPaging", "error", err)
		http.Error(w, "Could not search ldap server", http.StatusInternalServerError)
		return
	}
	if len(res.Entries) == 0 {
		http.Error(w, "No such user", http.StatusNotFound)
		return
	}
	if len(res.Entries) > 1 {
		slog.Warn("Found multiple entries", "kthid", kthid, "count", len(res.Entries))
	}
	entry := res.Entries[0]
	var user struct {
		KTHID     string `ldap:"ugUsername" json:"kthid"`
		UGKTHID   string `ldap:"ugKthid"    json:"ug_kthid"`
		FirstName string `ldap:"givenName"  json:"first_name"`
		Surname   string `ldap:"sn"         json:"surname"`

		DisplayName string `ldap:"displayName" json:"-"`
		Email       string `ldap:"mail"        json:"-"`
		Cn          string `ldap:"cn"          json:"-"`
	}
	if err := entry.Unmarshal(&user); err != nil {
		slog.Error("Failed unmarshaling user", "kthid", kthid, "user", entry.Attributes, "error", err)
		http.Error(w, "Could not parse user from ldap server", http.StatusInternalServerError)
		return
	}
	if user.DisplayName != user.FirstName+" "+user.Surname ||
		user.Email != user.KTHID+"@kth.se" ||
		user.Cn != user.FirstName+" "+user.Surname+" ("+user.KTHID+")" {
		u, _ := json.MarshalIndent(user, "", "    ")
		slog.Warn("User doesn't match expectations", "user", string(u))
	}
	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(user); err != nil {
		slog.Error("Could not send user back", "error", err)
		http.Error(w, "Could not send user", http.StatusInternalServerError)
		return
	}
}
