package moynalog

import "context"

// MoyNalogService определяет интерфейс для отправки чеков.
type MoyNalogService interface {
    SendReceipt(ctx context.Context, data ReceiptData) error
    Auth(ctx context.Context) error
}