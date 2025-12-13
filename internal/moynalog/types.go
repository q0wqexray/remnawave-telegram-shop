package moynalog

import "time"

// ReceiptData содержит данные для отправки чека.
type ReceiptData struct {
    Amount        float64   // Сумма
    Description   string    // Описание услуги
    PaymentDate   time.Time // Дата платежа
    PaymentID     int       // Идентификатор платежа (для логов)
    CustomerEmail string    // Email клиента
    CustomerPhone string    // Телефон клиента
}