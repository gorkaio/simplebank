package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	mockdb "github.com/gorkaio/simplebank/db/mock"
	db "github.com/gorkaio/simplebank/db/sqlc"
	"github.com/gorkaio/simplebank/util"
	"github.com/stretchr/testify/require"
)

type eqCreateUserParamsMatcher struct {
	arg db.CreateUserParams
	password string
}

func (e eqCreateUserParamsMatcher) Matches(x interface{}) bool {
	arg, ok := x.(db.CreateUserParams)
	if !ok {
		return false
	}

	err := util.CheckPassword(e.password, arg.HashedPassword)
	if err != nil {
		return false
	}

	e.arg.HashedPassword = arg.HashedPassword
	return reflect.DeepEqual(e.arg, arg)
}

func (e eqCreateUserParamsMatcher) String() string {
	return fmt.Sprintf("matches arg %v and password %v", e.arg, e.password)
}

func EqCreateUserParams(arg db.CreateUserParams, password string) gomock.Matcher {
	return eqCreateUserParamsMatcher{arg, password}
}

func TestCreateUserAPI(t *testing.T) {
	user, password := randomUser(t)

	testCases := []struct{
		name string
		body gin.H
		buildStubs func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"username": user.Username,
				"full_name": user.FullName,
				"password": password,
				"email": user.Email,
			},
			buildStubs: func (store *mockdb.MockStore)  {
				args := db.CreateUserParams{
					Username: user.Username,
					FullName: user.FullName,
					HashedPassword: user.HashedPassword,
					Email: user.Email,
				}
				store.EXPECT().
					CreateUser(gomock.Any(), EqCreateUserParams(args, password)).
					Times(1).
					Return(user, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchCreateUser(t, recorder.Body, user)
			},
		},
		{
			name: "InternalServerError",
			body: gin.H{
				"username": user.Username,
				"full_name": user.FullName,
				"password": password,
				"email": user.Email,
			},
			buildStubs: func (store *mockdb.MockStore)  {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.User{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "BadRequestWithoutEmail",
			body: gin.H{
				"username": user.Username,
				"full_name": user.FullName,
				"password": password,
			},
			buildStubs: func (store *mockdb.MockStore)  {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Eq(gomock.Any())).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "BadRequestWithoutPassword",
			body: gin.H{
				"username": user.Username,
				"full_name": user.FullName,
				"email": user.Email,
			},
			buildStubs: func (store *mockdb.MockStore)  {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Eq(gomock.Any())).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "BadRequestWithoutUsername",
			body: gin.H{
				"full_name": user.FullName,
				"password": password,
				"email": user.Email,
			},
			buildStubs: func (store *mockdb.MockStore)  {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Eq(gomock.Any())).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "BadRequestWithoutFullname",
			body: gin.H{
				"username": user.Username,
				"password": password,
				"email": user.Email,
			},
			buildStubs: func (store *mockdb.MockStore)  {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Eq(gomock.Any())).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "BadRequestWithoutPasswordTooShort",
			body: gin.H{
				"username": user.Username,
				"full_name": user.FullName,
				"password": "123",
				"email": user.Email,
			},
			buildStubs: func (store *mockdb.MockStore)  {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Eq(gomock.Any())).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
		
			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)
		
			server := NewServer(store)
			recorder := httptest.NewRecorder()

			body, err := json.Marshal(tc.body)
			require.NoError(t, err)

			request, err := http.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(body))
			require.NoError(t, err)
		
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func randomUser(t *testing.T) (user db.User, password string) {
	password = util.RandomString(6)
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	user = db.User{
		Username: util.RandomOwner(),
		FullName:    util.RandomOwner(),
		Email:  util.RandomEmail(),
		HashedPassword: hashedPassword,
	}

	return
}

func requireBodyMatchCreateUser(t *testing.T, body *bytes.Buffer, user db.User) {
	data, err := ioutil.ReadAll(body)
	require.NoError(t, err)

	var response createUserResponse
	err = json.Unmarshal(data, &response)
	require.NoError(t, err)
	require.Equal(t, response, createUserResponse{
		Username: user.Username,
		FullName: user.FullName,
		Email: user.Email,
		CreatedAt: user.CreatedAt,
	})
}