package moynalog

import (
    "context"
    "fmt"
    "os"
    "log"
    "time"
    "github.com/shoman4eg/go-moy-nalog/moynalog"
    "github.com/shopspring/decimal"
)

// moyNalogClient реализует интерфейс MoyNalogService
type moyNalogClient struct {
    client *moynalog.Client
}

// New создает новый экземпляр клиента
func New() (*moyNalogClient, error) {
    login := os.Getenv("MOY_NALOG_LOGIN")
    password := os.Getenv("MOY_NALOG_PASSWORD")

    if login == "" || password == "" {
        return nil, fmt.Errorf("MOY_NALOG_LOGIN and MOY_NALOG_PASSWORD environment variables must be set")
    }

    // Создаем клиент без токена
    client := moynalog.NewClient(nil)

    // Авторизуемся и получаем токен
    token, err := client.Auth.CreateAccessToken(context.Background(), login, password)
    if err != nil {
        return nil, fmt.Errorf("failed to create access token: %w", err)
    }

    // Создаем новый клиент с токеном
    authenticatedClient := moynalog.NewAuthClient(token)

    return &moyNalogClient{
        client: authenticatedClient,
    }, nil
}

// SendReceipt отправляет чек в "Мой налог"
func (c *moyNalogClient) SendReceipt(ctx context.Context, data ReceiptData) error {
    // Преобразуем сумму в строку с двумя знаками после запятой
    amountStr := fmt.Sprintf("%.2f", data.Amount)

    // Создаем элемент услуги
    serviceItem := &moynalog.IncomeServiceItem{
        Name:     data.Description,
        Amount:   decimal.NewFromFloat(data.Amount),
        Quantity: 1,
    }

    // Гарантируем, что дата установлена и не равна нулю
    requestTime := data.PaymentDate
    if requestTime.IsZero() {
        requestTime = time.Now().UTC()
    }

    // Создаем запрос на отправку чека
    incomeRequest := &moynalog.IncomeCreateRequest{
        PaymentType:   moynalog.Cash, // или другой подходящий тип оплаты
        RequestTime:   requestTime,
        OperationTime: requestTime,
        Services:      []*moynalog.IncomeServiceItem{serviceItem},
        TotalAmount:   amountStr,
        IgnoreMaxTotalIncomeRestriction: false,
    }

    // Добавляем информацию о клиенте если она доступна
    if data.CustomerEmail != "" || data.CustomerPhone != "" {
        // Используем правильное имя структуры IncomeClient
        incomeRequest.Client = &moynalog.IncomeClient{
            ContactPhone: data.CustomerPhone,
            DisplayName:  data.CustomerEmail, // Используем email как отображаемое имя
        }
    } else {
        // Устанавливаем минимальную информацию о клиенте, чтобы избежать nil-указателя
        incomeRequest.Client = &moynalog.IncomeClient{
            DisplayName: "Unknown Customer",
        }
    }

    // Отправляем чек
    _, err := c.client.Income.Create(ctx, incomeRequest)
    if err != nil {
        log.Printf("Failed to send receipt to Moy Nalog: %v, PaymentID: %d, Amount: %.2f",
            err, data.PaymentID, data.Amount)
        // Ошибка логируется, но не вызывает панику и не прерывает основной процесс
        return nil // Возвращаем nil, чтобы избежать паники при отправке чека
    }

    log.Printf("Successfully sent receipt to Moy Nalog: PaymentID: %d, Amount: %.2f",
        data.PaymentID, data.Amount)
    return nil
}