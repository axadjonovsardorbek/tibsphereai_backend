package models

// Payme JSON-RPC 2.0 protokoli (checkout.paycom.uz webhook: POST /payme).
// Referens: furasentr-payment/internal/api/handlers/payme.go.

const (
	PaymeErrorMethodNotFound          = -32601
	PaymeErrorAuth                    = -32504
	PaymeErrorInsufficientPrivileges  = -32504
	PaymeErrorInvalidAmount           = -31001
	PaymeErrorTransactionNotFound     = -31008
	PaymeErrorAlreadyPaid             = -31051
	PaymeErrorInternal                = -32400
	PaymeErrorCantCancel              = -31007
	PaymeErrorAccountNotFound         = -31050
	PaymeErrorOtherTransactionPending = -31099
)

// Payme transaction state (docs: PAYMENT_TRANSACTIONS_SCHEMA.sql "state" ustuni bilan bir xil ma'no).
const (
	PaymeStateCreated   = 1
	PaymeStatePerformed = 2
	PaymeStateCancelled = -1
	PaymeStateRefunded  = -2
)

type PaymeRequest struct {
	Method string         `json:"method"`
	Params map[string]any `json:"params"`
	ID     any            `json:"id"`
}

type PaymeResponse struct {
	JsonRPC string         `json:"jsonrpc"`
	ID      any            `json:"id"`
	Result  map[string]any `json:"result,omitempty"`
	Error   *PaymeError    `json:"error,omitempty"`
}

type PaymeError struct {
	Code    int            `json:"code"`
	Message map[string]any `json:"message"` // Payme spec: {uz, ru, en}
	Data    string         `json:"data,omitempty"`
}

// PaymeOpError - protokol darajasidagi rad javoblari (account not found, invalid amount,
// already paid, ...) uchun storage/service qatlamidan qaytariladi; handler shuni
// errors.As bilan ushlab, aynan shu Code/Message/Data bilan JSON-RPC error qaytaradi.
// Oddiy (DB/internal) xatolar bunga o'ralmaydi - ular generic PaymeErrorInternal bo'ladi.
type PaymeOpError struct {
	Code    int
	Message string
	Data    string
}

func (e *PaymeOpError) Error() string { return e.Message }

// Helpers to extract params (spf13/cast o'rniga oddiy type-assertion, tashqi bog'liqlik qo'shmaslik uchun).
func (r *PaymeRequest) GetParamString(key string) string {
	if v, ok := r.Params[key].(string); ok {
		return v
	}
	return ""
}

func (r *PaymeRequest) GetParamInt64(key string) int64 {
	switch v := r.Params[key].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	}
	return 0
}

func (r *PaymeRequest) GetAccount() map[string]any {
	if v, ok := r.Params["account"].(map[string]any); ok {
		return v
	}
	return map[string]any{}
}

// ============ Storage qatlami uchun DTO'lar ============

type PaymeCreateResult struct {
	OrderId     string
	CreateTime  int64
	PerformTime int64
	CancelTime  int64
	State       int
	Reason      *int
}

type PaymeTransactionResult struct {
	OrderId     string
	CreateTime  int64
	PerformTime int64
	CancelTime  int64
	State       int
	Reason      *int
	// JustPaid - PerformTransaction birinchi marta state=1->2 o'tganda true (service shu orqali
	// promokod/push logikasini ishga tushiradi, ApplyComplete/Click bilan bir xil pattern).
	JustPaid bool
}

type PaymeStatementItem struct {
	OrderId     string
	PaymeId     string
	Amount      float64
	CreateTime  int64
	PerformTime int64
	CancelTime  int64
	State       int
	Reason      *int
}
