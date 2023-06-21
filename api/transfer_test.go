package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	mockdb "github.com/gorkaio/simplebank/db/mock"
	db "github.com/gorkaio/simplebank/db/sqlc"
	"github.com/gorkaio/simplebank/util"
	"github.com/stretchr/testify/require"
)

func TestCreateTransferAPI(t *testing.T) {
	currency := util.EUR
	account_from := randomAccountWithCurrency(currency)
	account_to := randomAccountWithCurrency(currency)
	transfer, entry_from, entry_to := randomTransferForAccounts(account_from.ID, account_to.ID)

	testCases := []struct {
		name          string
		body      gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"from_account_id": account_from.ID,
				"to_account_id": account_to.ID,
				"currency": currency,
				"amount": transfer.Amount,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), account_from.ID).
					Times(1).
					Return(account_from, nil)
				store.EXPECT().
					GetAccount(gomock.Any(), account_to.ID).
					Times(1).
					Return(account_to, nil)
				store.EXPECT().
					CreateTransferTx(gomock.Any(), gomock.Eq(db.CreateTransferTxParams{
						FromAccountID: transfer.FromAccountID,
						ToAccountID:   transfer.ToAccountID,
						Amount:        transfer.Amount,
					})).
					Times(1).
					Return(db.CreateTransferTxResult{
						Transfer:    transfer,
						FromAccount: account_from,
						ToAccount:   account_to,
						FromEntry:   entry_from,
						ToEntry:     entry_to,
					}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, recorder.Code, http.StatusOK)
				requireBodyMatchTransfer(t, recorder.Body, transfer)
			},
		},
		{
			name: "BadRequest",
			body: gin.H{
				"currency": currency,
				"amount": transfer.Amount,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Any()).
					Times(0)
				store.EXPECT().
					CreateTransferTx(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, recorder.Code, http.StatusBadRequest)
			},
		},
		{
			name: "InternalErrorOnValidation",
			body: gin.H{
				"from_account_id": account_from.ID,
				"to_account_id": account_to.ID,
				"currency": currency,
				"amount": transfer.Amount,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), account_from.ID).
					Times(1).
					Return(db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, recorder.Code, http.StatusInternalServerError)
			},
		},
		{
			name: "InternalErrorOnTx",
			body: gin.H{
				"from_account_id": account_from.ID,
				"to_account_id": account_to.ID,
				"currency": currency,
				"amount": transfer.Amount,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), account_from.ID).
					Times(1).
					Return(account_from, nil)
				store.EXPECT().
					GetAccount(gomock.Any(), account_to.ID).
					Times(1).
					Return(account_to, nil)
				store.EXPECT().
					CreateTransferTx(gomock.Any(), gomock.Eq(db.CreateTransferTxParams{
						FromAccountID: transfer.FromAccountID,
						ToAccountID:   transfer.ToAccountID,
						Amount:        transfer.Amount,
					})).
					Times(1).
					Return(db.CreateTransferTxResult{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, recorder.Code, http.StatusInternalServerError)
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

			request, err := http.NewRequest(http.MethodPost, "/transfers", bytes.NewBuffer(body))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestListTransfers(t *testing.T) {
	transfers := []db.Transfer{}
	for i := 0; i < 10; i++ {
		transfer, _, _ := randomTransfer()
		transfers = append(transfers, transfer)
	}

	testCases := []struct {
		name          string
		url           string
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			url:  "/transfers?page_id=1&page_size=10",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListTranfers(gomock.Any(), gomock.Eq(db.ListTranfersParams{FromAccountID: 0, ToAccountID: 0, Limit: 10, Offset: 0})).
					Times(1).
					Return(transfers, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, recorder.Code, http.StatusOK)
				requireBodyMatchTransfers(t, recorder.Body, transfers)
			},
		},
		{
			name: "FromAccountOK",
			url:  "/transfers?page_id=1&page_size=10&from_account_id=" + fmt.Sprint(transfers[0].FromAccountID),
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListTranfers(gomock.Any(), gomock.Eq(db.ListTranfersParams{FromAccountID: transfers[0].FromAccountID, ToAccountID: 0, Limit: 10, Offset: 0})).
					Times(1).
					Return([]db.Transfer{transfers[0]}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, recorder.Code, http.StatusOK)
				requireBodyMatchTransfers(t, recorder.Body, []db.Transfer{transfers[0]})
			},
		},
		{
			name: "FromToAccountOK",
			url:  "/transfers?page_id=1&page_size=10&from_account_id=" + fmt.Sprint(transfers[0].FromAccountID) + "&to_account_id=" + fmt.Sprint(transfers[0].ToAccountID),
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListTranfers(gomock.Any(), gomock.Eq(db.ListTranfersParams{FromAccountID: transfers[0].FromAccountID, ToAccountID: transfers[0].ToAccountID, Limit: 10, Offset: 0})).
					Times(1).
					Return([]db.Transfer{transfers[0]}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, recorder.Code, http.StatusOK)
				requireBodyMatchTransfers(t, recorder.Body, []db.Transfer{transfers[0]})
			},
		},
		{
			name: "FromAccountOK",
			url:  "/transfers?page_id=1&page_size=10&to_account_id=" + fmt.Sprint(transfers[0].ToAccountID),
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListTranfers(gomock.Any(), gomock.Eq(db.ListTranfersParams{FromAccountID: 0, ToAccountID: transfers[0].ToAccountID, Limit: 10, Offset: 0})).
					Times(1).
					Return([]db.Transfer{transfers[0]}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, recorder.Code, http.StatusOK)
				requireBodyMatchTransfers(t, recorder.Body, []db.Transfer{transfers[0]})
			},
		},
		{
			name: "InternalError",
			url:  "/transfers?page_id=1&page_size=10",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListTranfers(gomock.Any(), gomock.Any()).
					Times(1).
					Return([]db.Transfer{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, recorder.Code, http.StatusInternalServerError)
			},
		},
		{
			name: "BadRequestWithoutPageId",
			url:  "/transfers?page_size=10",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListTranfers(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, recorder.Code, http.StatusBadRequest)
			},
		},
		{
			name: "BadRequestWithoutPageSize",
			url:  "/transfers?page_id=1",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, recorder.Code, http.StatusBadRequest)
			},
		},
		{
			name: "BadRequestWithoutParams",
			url:  "/transfers",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, recorder.Code, http.StatusBadRequest)
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

			url := tc.url
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestGetTransferAPI(t *testing.T) {
	transfer, _, _ := randomTransfer()

	testCases := []struct {
		name          string
		transferID    int64
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:       "OK",
			transferID: transfer.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransfer(gomock.Any(), gomock.Eq(transfer.ID)).
					Times(1).
					Return(transfer, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, recorder.Code, http.StatusOK)
				requireBodyMatchTransfer(t, recorder.Body, transfer)
			},
		},
		{
			name:       "NotFound",
			transferID: transfer.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransfer(gomock.Any(), gomock.Eq(transfer.ID)).
					Times(1).
					Return(db.Transfer{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, recorder.Code, http.StatusNotFound)
			},
		},
		{
			name:       "InternalError",
			transferID: transfer.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransfer(gomock.Any(), gomock.Eq(transfer.ID)).
					Times(1).
					Return(db.Transfer{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, recorder.Code, http.StatusInternalServerError)
			},
		},
		{
			name:       "BadRequest",
			transferID: 0,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTransfer(gomock.Any(), gomock.Eq(gomock.Any())).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, recorder.Code, http.StatusBadRequest)
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

			url := fmt.Sprintf("/transfers/%d", tc.transferID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func randomTransfer() (transfer db.Transfer, entry_from db.Entry, entry_to db.Entry) {
	transfer, entry_from, entry_to = randomTransferForAccounts(util.RandomInt(1, 1000), util.RandomInt(1, 1000))
	return
}

func randomTransferForAccounts(account_from_id int64, account_to_id int64) (transfer db.Transfer, entry_from db.Entry, entry_to db.Entry) {
	amount := util.RandomMoney()
	transfer = db.Transfer{
		ID:            util.RandomInt(1, 1000),
		FromAccountID: account_from_id,
		ToAccountID:   account_to_id,
		Amount:        amount,
	}
	entry_from = db.Entry{
		ID:        util.RandomInt(1, 1000),
		AccountID: account_from_id,
		Amount:    -amount,
	}
	entry_to = db.Entry{
		ID:        util.RandomInt(1, 1000),
		AccountID: account_to_id,
		Amount:    amount,
	}

	return
}

func randomAccountWithCurrency(currency string) db.Account {
	return db.Account{
		ID:       util.RandomInt(1, 1000),
		Owner:    util.RandomOwner(),
		Balance:  util.RandomMoney(),
		Currency: currency,
	}
}

func requireBodyMatchTransfer(t *testing.T, body *bytes.Buffer, transfer db.Transfer) {
	data, err := ioutil.ReadAll(body)
	require.NoError(t, err)

	var gotTransfer db.Transfer
	err = json.Unmarshal(data, &gotTransfer)
	require.NoError(t, err)
	require.Equal(t, transfer, gotTransfer)
}

func requireBodyMatchTransfers(t *testing.T, body *bytes.Buffer, transfers []db.Transfer) {
	data, err := ioutil.ReadAll(body)
	require.NoError(t, err)

	var gotTransfers []db.Transfer
	err = json.Unmarshal(data, &gotTransfers)
	require.NoError(t, err)
	require.Equal(t, transfers, gotTransfers)
}
