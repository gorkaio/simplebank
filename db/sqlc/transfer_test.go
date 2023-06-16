package db

import (
	"context"
	"testing"

	"github.com/gorkaio/simplebank/util"
	"github.com/stretchr/testify/require"
)

func createRandomTransfer(t *testing.T) Transfer {
	account_from := createRandomAccount(t)
	account_to := createRandomAccount(t)

	return createRandomTransferForAccounts(t, account_from, account_to)
}

func createRandomTransferForAccounts(t *testing.T, account_from Account, account_to Account) Transfer {
	arg := CreateTransferParams{
		FromAccountID: account_from.ID,
		ToAccountID: account_to.ID,
		Amount: util.RandomMoney(),
	}

	transfer, err := testQueries.CreateTransfer(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, transfer)

	require.Equal(t, transfer.FromAccountID, arg.FromAccountID)
	require.Equal(t, transfer.ToAccountID, arg.ToAccountID)
	require.Equal(t, transfer.Amount, arg.Amount)

	require.NotZero(t, transfer.ID)
	require.NotZero(t, transfer.CreatedAt)

	return transfer
}

func TestCreateTransfer(t *testing.T) {
	createRandomTransfer(t)
}

func TestGetTransfer(t *testing.T) {
	transfer1 := createRandomTransfer(t)

	transfer2, err := testQueries.GetTransfer(context.Background(), transfer1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, transfer2)

	require.Equal(t, transfer2.ID, transfer1.ID)
	require.Equal(t, transfer2.FromAccountID, transfer1.FromAccountID)
	require.Equal(t, transfer2.ToAccountID, transfer1.ToAccountID)
	require.Equal(t, transfer2.Amount, transfer1.Amount)
	require.Equal(t, transfer2.CreatedAt, transfer1.CreatedAt)
}

func TestListTransfers(t *testing.T) {
	for i := 0; i < 10; i++ {
		createRandomTransfer(t)
	}

	arg := ListTranfersParams{
		Limit: 5,
		Offset: 5,
	}

	transfers, err := testQueries.ListTranfers(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, transfers, 5)

	for _, transfer := range transfers {
		require.NotEmpty(t, transfer)
	}
}

func TestListTransfersFromAccount(t *testing.T) {
	for i := 0; i < 10; i++ {
		createRandomTransfer(t)
	}

	account_from := createRandomAccount(t)
	for i := 0; i < 10; i++ {
		account_to := createRandomAccount(t)
		createRandomTransferForAccounts(t, account_from, account_to)
	}

	arg := ListTranfersParams{
		FromAccountID: account_from.ID,
		Limit: 5,
		Offset: 5,
	}

	transfers, err := testQueries.ListTranfers(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, transfers, 5)

	for _, transfer := range transfers {
		require.NotEmpty(t, transfer)
		require.Equal(t, transfer.FromAccountID, account_from.ID)
	}
}

func TestListTransfersToAccount(t *testing.T) {
	for i := 0; i < 10; i++ {
		createRandomTransfer(t)
	}

	account_to := createRandomAccount(t)
	for i := 0; i < 10; i++ {
		account_from := createRandomAccount(t)
		createRandomTransferForAccounts(t, account_from, account_to)
	}

	arg := ListTranfersParams{
		ToAccountID: account_to.ID,
		Limit: 5,
		Offset: 5,
	}

	transfers, err := testQueries.ListTranfers(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, transfers, 5)

	for _, transfer := range transfers {
		require.NotEmpty(t, transfer)
		require.Equal(t, transfer.ToAccountID, account_to.ID)
	}
}

func TestListTransfersBetweenAccounts(t *testing.T) {
	for i := 0; i < 10; i++ {
		createRandomTransfer(t)
	}

	account_to := createRandomAccount(t)
	account_from := createRandomAccount(t)
	for i := 0; i < 10; i++ {
		createRandomTransferForAccounts(t, account_from, account_to)
	}

	arg := ListTranfersParams{
		FromAccountID: account_from.ID,
		ToAccountID: account_to.ID,
		Limit: 5,
		Offset: 5,
	}

	transfers, err := testQueries.ListTranfers(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, transfers, 5)

	for _, transfer := range transfers {
		require.NotEmpty(t, transfer)
		require.Equal(t, transfer.ToAccountID, account_to.ID)
	}
}