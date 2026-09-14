package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Error.Code
}

func TestHandlerRegisterLoginRefreshMeLogout(t *testing.T) {
	h := NewHandler(testService(newMemStore()), false)

	reg := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"a@b.com","password":"password1"}`))
	req.Header.Set("Content-Type", "application/json")
	h.Register(reg, req)
	if reg.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", reg.Code, reg.Body.String())
	}
	if len(reg.Result().Cookies()) != 1 {
		t.Fatal("expected refresh cookie")
	}

	login := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"a@b.com","password":"password1"}`))
	req.Header.Set("Content-Type", "application/json")
	h.Login(login, req)
	if login.Code != http.StatusOK {
		t.Fatalf("login: %d %s", login.Code, login.Body.String())
	}
	var sess sessionBody
	if err := json.Unmarshal(login.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}
	cookie := login.Result().Cookies()[0]

	me := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+sess.AccessToken)
	h.Bearer(http.HandlerFunc(h.Me)).ServeHTTP(me, req)
	if me.Code != http.StatusOK {
		t.Fatalf("me: %d %s", me.Code, me.Body.String())
	}

	ref := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(cookie)
	h.Refresh(ref, req)
	if ref.Code != http.StatusOK {
		t.Fatalf("refresh: %d %s", ref.Code, ref.Body.String())
	}

	out := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(ref.Result().Cookies()[0])
	req.Header.Set("Authorization", "Bearer "+sess.AccessToken)
	h.Logout(out, req)
	if out.Code != http.StatusNoContent {
		t.Fatalf("logout: %d", out.Code)
	}
}

func TestHandlerErrorCodes(t *testing.T) {
	h := NewHandler(testService(newMemStore()), false)

	bad := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"nope","password":"short"}`))
	req.Header.Set("Content-Type", "application/json")
	h.Register(bad, req)
	if bad.Code != http.StatusBadRequest || errorCode(t, bad) != "invalid_request" {
		t.Fatalf("invalid register: %d %s", bad.Code, bad.Body.String())
	}

	ok := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"a@b.com","password":"password1"}`))
	req.Header.Set("Content-Type", "application/json")
	h.Register(ok, req)
	if ok.Code != http.StatusCreated {
		t.Fatal(ok.Body.String())
	}

	dup := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"a@b.com","password":"password1"}`))
	req.Header.Set("Content-Type", "application/json")
	h.Register(dup, req)
	if dup.Code != http.StatusConflict || errorCode(t, dup) != "email_taken" {
		t.Fatalf("dup: %d %s", dup.Code, dup.Body.String())
	}

	wrong := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"a@b.com","password":"password2"}`))
	req.Header.Set("Content-Type", "application/json")
	h.Login(wrong, req)
	if wrong.Code != http.StatusUnauthorized || errorCode(t, wrong) != "invalid_credentials" {
		t.Fatalf("wrong: %d %s", wrong.Code, wrong.Body.String())
	}

	unauth := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	h.Bearer(http.HandlerFunc(h.Me)).ServeHTTP(unauth, req)
	if unauth.Code != http.StatusUnauthorized || errorCode(t, unauth) != "unauthorized" {
		t.Fatalf("me: %d %s", unauth.Code, unauth.Body.String())
	}
}
