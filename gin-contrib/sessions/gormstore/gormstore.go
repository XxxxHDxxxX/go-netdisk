/*
Package gormstore is a GORM backend for gorilla sessions

Simplest form:

	store := gormstore.New(gorm.Open(...), []byte("secret-hash-key"))

All options:

	store := gormstore.NewGormStore(
		gorm.Open(...), // *gorm.DB
		gormstore.Options{
			TableName: "sessions",  // "sessions" is default
			SkipCreateTable: false, // false is default
		},
		[]byte("secret-hash-key"),       // 32 or 64 bytes recommended, required
		[]byte("secret-encryption-key")) // nil, 16, 24 or 32 bytes, optional

		// some more settings, see sessions.sessionOptions
		store.sessionOptions.Secure = true
		store.sessionOptions.HttpOnly = true
		store.sessionOptions.MaxAge = 60 * 60 * 24 * 60

If you want periodic cleanup of expired sessions:

		quit := make(chan struct{})
		go store.PeriodicCleanup(1*time.Hour, quit)

For more information about the keys see https://github.com/gorilla/securecookie

For API to use in HTTP handlers see https://github.com/gorilla/sessions
*/
package gormstore

import (
	"encoding/base32"
	"log"
	"net/http"
	"strings"
	"time"

	ginsessions "github.com/gin-contrib/sessions"
	"github.com/gorilla/context"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

const sessionIDLen = 32
const defaultTableName = "sessions"
const defaultMaxAge = 60 * 60 * 24 * 30 // 30 days
const defaultPath = "/"

// Options for gormstore
type Options struct {
	TableName       string
	SkipCreateTable bool
}

// GormStore store sessions in db by gorm
type GormStore struct {
	db             *gorm.DB
	opts           Options
	Codecs         []securecookie.Codec
	sessionOptions *sessions.Options
}

type gormSession struct {
	ID        string `gorm:"unique_index"`
	Data      string `gorm:"type:text"`
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt time.Time `gorm:"index"`
}

// Define a type for context keys so that they can't clash with anything else stored in context
type contextKey string

// New creates a new gormstore session
func New(db *gorm.DB, keyPairs ...[]byte) *GormStore {
	return NewGormStore(db, Options{}, keyPairs...)
}

// NewGormStore creates a new gormstore session with options
func NewGormStore(db *gorm.DB, opts Options, keyPairs ...[]byte) *GormStore {
	st := &GormStore{
		db:     db,
		opts:   opts,
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		sessionOptions: &sessions.Options{
			Path:   defaultPath,
			MaxAge: defaultMaxAge,
		},
	}
	if st.opts.TableName == "" {
		st.opts.TableName = defaultTableName
	}

	if !st.opts.SkipCreateTable {
		st.sessionTable().AutoMigrate(&gormSession{})
	}

	return st
}

func (st *GormStore) Options(options ginsessions.Options) {
	st.sessionOptions = options.ToGorillaOptions()
}

func (st *GormStore) sessionTable() *gorm.DB {
	return st.db.Table(st.opts.TableName)
}

// Get returns a session for the given name after adding it to the registry.
func (st *GormStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(st, name)
}

// New creates a session with name without adding it to the registry.
func (st *GormStore) New(r *http.Request, name string) (*sessions.Session, error) {
	session := sessions.NewSession(st, name)
	opts := *st.sessionOptions
	session.Options = &opts

	st.MaxAge(st.sessionOptions.MaxAge)

	// try fetch from db if there is a cookie
	if cookie, err := r.Cookie(name); err == nil {
		if err := securecookie.DecodeMulti(name, cookie.Value, &session.ID, st.Codecs...); err != nil {
			return session, nil
		}
		s := &gormSession{}
		sr := st.sessionTable().Where("id = ? AND expires_at > ?", session.ID, time.Now()).Limit(1).Find(s)
		if sr.Error != nil || sr.RowsAffected == 0 {
			return session, nil
		}
		if err := securecookie.DecodeMulti(session.Name(), s.Data, &session.Values, st.Codecs...); err != nil {
			return session, nil
		}

		context.Set(r, contextKey(name), s)
	}

	return session, nil
}

// Save session and set cookie header
func (st *GormStore) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	s, _ := context.Get(r, contextKey(session.Name())).(*gormSession)

	// delete if max age is < 0
	if session.Options.MaxAge < 0 {
		if s != nil {
			if err := st.sessionTable().Delete(s).Error; err != nil {
				return err
			}
		}
		http.SetCookie(w, sessions.NewCookie(session.Name(), "", session.Options))
		return nil
	}

	data, err := securecookie.EncodeMulti(session.Name(), session.Values, st.Codecs...)
	if err != nil {
		return err
	}
	now := time.Now()
	expire := now.Add(time.Second * time.Duration(session.Options.MaxAge))

	if s == nil {
		// generate random session ID key suitable for storage in the db
		session.ID = strings.TrimRight(
			base32.StdEncoding.EncodeToString(
				securecookie.GenerateRandomKey(sessionIDLen)), "=")
		s = &gormSession{
			ID:        session.ID,
			Data:      data,
			CreatedAt: now,
			UpdatedAt: now,
			ExpiresAt: expire,
		}
		if err := st.sessionTable().Create(s).Error; err != nil {
			return err
		}
		context.Set(r, contextKey(session.Name()), s)
	} else {
		s.Data = data
		s.UpdatedAt = now
		s.ExpiresAt = expire
		if err := st.sessionTable().Save(s).Error; err != nil {
			return err
		}
	}

	// set session id cookie
	id, err := securecookie.EncodeMulti(session.Name(), session.ID, st.Codecs...)
	if err != nil {
		return err
	}
	http.SetCookie(w, sessions.NewCookie(session.Name(), id, session.Options))

	return nil
}

// MaxAge sets the maximum age for the store and the underlying cookie
// implementation. Individual sessions can be deleted by setting
// sessionOptions.MaxAge = -1 for that session.
func (st *GormStore) MaxAge(age int) {
	st.sessionOptions.MaxAge = age
	for _, codec := range st.Codecs {
		if sc, ok := codec.(*securecookie.SecureCookie); ok {
			sc.MaxAge(age)
		}
	}
}

// MaxLength restricts the maximum length of new sessions to l.
// If l is 0 there is no limit to the size of a session, use with caution.
// The default is 4096 (default for securecookie)
func (st *GormStore) MaxLength(l int) {
	for _, c := range st.Codecs {
		if codec, ok := c.(*securecookie.SecureCookie); ok {
			codec.MaxLength(l)
		}
	}
}

// Cleanup deletes expired sessions
func (st *GormStore) Cleanup() {
	log.Printf("Session clean up: %s\n", time.Now())
	st.sessionTable().Delete(&gormSession{}, "expires_at <= ?", time.Now())
}

// PeriodicCleanup runs Cleanup every interval. Close quit channel to stop.
func (st *GormStore) PeriodicCleanup(interval time.Duration, quit <-chan struct{}) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			st.Cleanup()
		case <-quit:
			return
		}
	}
}
