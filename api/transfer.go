package api

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/gorkaio/simplebank/db/sqlc"
)

type createTransferRequest struct {
	FromAccountID int64 `json:"from_account_id" binding:"required,min=1"`
	ToAccountID int64 `json:"to_account_id" binding:"required,min=1"`
	Amount int64 `json:"amount" binding:"required,gt=0"`
	Currency string `json:"currency" binding:"required,currency"`
}

func (server *Server) createTransfer(ctx *gin.Context) {
	var req createTransferRequest
	if err := ctx.ShouldBindJSON(&req); err != nil  {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if !server.validAccount(ctx, req.FromAccountID, req.Currency) || !server.validAccount(ctx, req.ToAccountID, req.Currency) {
		return
	}

	arg := db.TransferTxParams{
		FromAccountID: req.FromAccountID,
		ToAccountID: req.ToAccountID,
		Amount: req.Amount,
	}

	result, err := server.store.TransferTx(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, result)
}

type getTransferRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) getTransfer(ctx *gin.Context) {
	var req getTransferRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	transfer, err := server.store.GetTransfer(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, transfer)
}

type listTransfersRequest struct {
	PageID int32 `form:"page_id" binding:"required,min=1`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=10`
}

func (server *Server) listTransfers(ctx *gin.Context) {
	var req listTransfersRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	
	transfers, err := server.store.ListTranfers(ctx, db.ListTranfersParams{
		Limit: req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	})
	
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, transfers)
}

func (server *Server) validAccount(ctx *gin.Context, accountID int64, currency string) bool {
	account, err := server.store.GetAccount(ctx, accountID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return false
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return false
	}

	if account.Currency != currency {
		err := fmt.Errorf("account [%d] currency mismatch: %s vs %s", accountID, account.Currency, currency)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return false
	}

	return true
}