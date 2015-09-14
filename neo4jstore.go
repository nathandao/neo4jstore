package neo4jstore

import (
	"encoding/base32"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/jmcvetta/neoism"
)

// Neo4jStore represents the currently configured session store
type Neo4jStore struct {
	Db      *neoism.Database
	Options *sessions.Options
	Codecs  []securecookie.Codec
}

type Session struct {
	Id        int64     `json:"id(s)"`
	Key       string    `json:"u.key"`
	Data      string    `json:"u.data"`
	CreatedOn time.Time `json:"u.created_on"`
	ExpiresOn time.Time `json:"u.expires_on"`
}

// Options type
type Options struct {
	Path     string `json:"o.path"`
	Domain   string `json:"o.domain"`
	MaxAge   int    `json:"o.max_age"`
	Secure   bool   `json:"o.secure"`
	HttpOnly bool   `json:"o.http_only"`
}

// NewNeo4jStore creates a new Neo4jStore
func NewNeo4jStore(db *neoism.Database, keyPairs ...[]byte) *Neo4jStore {
	cs := &Neo4jStore{
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		Options: &sessions.Options{
			Path:   "/",
			MaxAge: 86400 * 30,
		},
		Db: db,
	}
	cs.MaxAge(cs.Options.MaxAge)
	return cs
}

func (n *Neo4jStore) MaxAge(age int) {
	n.Options.MaxAge = age
	// Set the maxAge for each securecookie instance.
	for _, codec := range n.Codecs {
		if sc, ok := codec.(*securecookie.SecureCookie); ok {
			sc.MaxAge(age)
		}
	}
}

// New creates a new session
func (n *Neo4jStore) New(r *http.Request, name string) (
	*sessions.Session, error) {
	session := sessions.NewSession(n, name)
	if session == nil {
		return session, nil
	}
	opts := *n.Options
	session.Options = &(opts)
	session.IsNew = true

	var err error
	if c, errCookie := r.Cookie(name); errCookie == nil {
		err = securecookie.DecodeMulti(name, c.Value, &session.ID, n.Codecs...)
		if err == nil {
			err = n.load(session)
			if err == nil {
				session.IsNew = false
			}
		}
	}
	n.MaxAge(n.Options.MaxAge)
	return session, err
}

// Get Fetches a session for a given name after it has been added to the
// registry.
func (n *Neo4jStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(n, name)
}

// Save adds a session to the database
func (n *Neo4jStore) Save(
	r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	// delete cookie if MaxAge < 0
	if session.Options.MaxAge < 0 {
		if err := n.destroy(session); err != nil {
			return err
		}
		// set empty cookie
		http.SetCookie(w, sessions.NewCookie(session.Name(), "", session.Options))
		return nil
	}

	if session.ID == "" {
		// Generate a random session ID key suitable for storage in the DB
		session.ID = strings.TrimRight(
			base32.StdEncoding.EncodeToString(
				securecookie.GenerateRandomKey(32)), "=")
	}

	if err := n.save(session); err != nil {
		return err
	}

	// Keep the session ID key in a cookie so it can be looked up in DB later.
	encoded, err := securecookie.EncodeMulti(session.Name(), session.ID, n.Codecs...)
	if err != nil {
		return err
	}

	http.SetCookie(w, sessions.NewCookie(session.Name(), encoded, session.Options))
	return nil
}

func (n *Neo4jStore) load(s *sessions.Session) error {
	r := []Session{}
	cq := neoism.CypherQuery{
		Statement: `MATCH (s:Session)
								WHERE s.key = {key}
								RETURN s.key, id(s), s.created_on, s.expires_on`,
		Parameters: neoism.Props{"key": s.ID},
		Result:     &r,
	}
	if err := n.Db.Cypher(&cq); err != nil {
		return err
	}
	if len(r) > 0 {
		return nil
	}
	return nil
}

func (n *Neo4jStore) destroy(s *sessions.Session) error {
	cq := neoism.CypherQuery{
		Statement: `MATCH (s:Session {key: {key}})
								OPTIONAL MATCH (s)-[r]->()
								DELETE s, r`,
		Parameters: neoism.Props{"key": s.ID},
	}
	if err := n.Db.Cypher(&cq); err != nil {
		return err
	}
	return nil
}

// save writes encoded session.Values to a database record.
// writes to http_sessions table by default.
func (n *Neo4jStore) save(session *sessions.Session) error {
	encoded, err := securecookie.EncodeMulti(session.Name(), session.Values, n.Codecs...)
	if err != nil {
		return err
	}
	var createdOn time.Time
	var expiresOn time.Time
	crOn := session.Values["created_on"]
	exOn := session.Values["expires_on"]
	if crOn == nil {
		createdOn = time.Now()
	} else {
		createdOn = crOn.(time.Time)
	}
	if exOn == nil {
		expiresOn = time.Now().Add(time.Second * time.Duration(session.Options.MaxAge))
	} else {
		expiresOn = exOn.(time.Time)
		if expiresOn.Sub(time.Now().Add(time.Second*time.Duration(session.Options.MaxAge))) < 0 {
			expiresOn = time.Now().Add(time.Second * time.Duration(session.Options.MaxAge))
		}
	}
	cq := neoism.CypherQuery{
		Statement: `CREATE (s:Session {
                  key: {key},
                  data: {data},
                  created_on: {created_on},
                  expires_on: {expires_on},
                  modifield_on: {modifield_on}
                })`,
		Parameters: neoism.Props{
			"key":          session.ID,
			"data":         encoded,
			"created_on":   createdOn,
			"expires_on":   expiresOn,
			"modifield_on": time.Now(),
		},
	}
	if err := n.Db.Cypher(&cq); err != nil {
		return err
	}
	return nil
}

// New returns a session for the given name without adding it to the registry.
// func (m *Neo4jStore) New(r *http.Request, name string) (
// 	*sessions.Session, error) {
// 	session := sessions.NewSession(m, name)
// 	session.Options = &sessions.Options{
// 		Path:   m.Options.Path,
// 		MaxAge: m.Options.MaxAge,
// 	}
// 	session.IsNew = true
// 	var err error
// 	if cook, errToken := m.Token.GetToken(r, name); errToken == nil {
// 		err = securecookie.DecodeMulti(name, cook, &session.ID, m.Codecs...)
// 		if err == nil {
// 			err = m.load(session)
// 			if err == nil {
// 				session.IsNew = false
// 			} else {
// 				err = nil
// 			}
// 		}
// 	}
// 	return session, err
// }

// MaxLength restricts the maximum length of new sessions to l.
func (n *Neo4jStore) MaxLength(l int) {
	for _, c := range n.Codecs {
		if codec, ok := c.(*securecookie.SecureCookie); ok {
			codec.MaxLength(l)
		}
	}
}
