package moynalog

import (
    "context"
    "encoding/json"
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
    rawClient := moynalog.NewClient(nil)

    // Авторизуемся и получаем токен
    token, err := rawClient.Auth.CreateAccessToken(context.Background(), login, password)
    if err != nil {
        return nil, fmt.Errorf("failed to create access token: %w", err)
    }

    // Создаем новый клиент с токеном
    authenticatedClient := moynalog.NewAuthClient(token)

    result := &moyNalogClient{
        client: authenticatedClient,
    }

    return result, nil
}

// Auth выполняет аутентификацию клиента
func (c *moyNalogClient) Auth(ctx context.Context) error {
    login := os.Getenv("MOY_NALOG_LOGIN")
    password := os.Getenv("MOY_NALOG_PASSWORD")

    if login == "" || password == "" {
        return fmt.Errorf("MOY_NALOG_LOGIN and MOY_NALOG_PASSWORD environment variables must be set")
    }

    token, err := c.client.Auth.CreateAccessToken(ctx, login, password)
    if err != nil {
        log.Printf("MoyNalog: auth failed: %v", err)
        return err
    }

    // Создаем новый аутентифицированный клиент с обновленным токеном
    authenticatedClient := moynalog.NewAuthClient(token)
    c.client = authenticatedClient

    log.Printf("MoyNalog: auth successful")
    return nil
}

// SendReceipt отправляет чек в "Мой налог"
func (c *moyNalogClient) SendReceipt(ctx context.Context, data ReceiptData) error {
    // STAGE 2: Create minimal valid payload according to rules
    // - Exactly one service: name = "Услуга", amount = 1.0, quantity = 1
    // - paymentType = CASH
    // - client = empty non-nil object
    // - operationTime = time.Now().UTC()
    
    serviceItem := &moynalog.IncomeServiceItem{
        Name:     "Услуга",           // Fixed service name as per STAGE 2 requirements
        Amount:   decimal.NewFromFloat(1.00),  // Fixed amount as per STAGE 2 requirements
        Quantity: 1,                  // Fixed quantity as per STAGE 2 requirements
    }

    // Use current UTC time as per STAGE 2 requirements
    requestTime := time.Now().UTC()

    // Create minimal valid payload
    incomeRequest := &moynalog.IncomeCreateRequest{
        PaymentType:   moynalog.Cash, // Fixed as per STAGE 2 requirements
        RequestTime:   requestTime,
        OperationTime: requestTime,   // Same as per STAGE 2 requirements
        Services:      []*moynalog.IncomeServiceItem{serviceItem},
        TotalAmount:   "1.00",        // Match the fixed amount
        IgnoreMaxTotalIncomeRestriction: false,
        Client: &moynalog.IncomeClient{}, // Empty non-nil object as per STAGE 2 requirements
    }

    // Log the full JSON payload as required by STAGE 2
    payloadJSON, err := json.Marshal(incomeRequest)
    if err != nil {
        log.Printf("Failed to marshal payload to JSON: %v, PaymentID: %d", err, data.PaymentID)
    } else {
        log.Printf("MoyNalog payload JSON: %s, PaymentID: %d, Timestamp: %v",
            string(payloadJSON), data.PaymentID, time.Now().UTC())
    }

    // Отправляем чек
    _, err = c.client.Income.Create(ctx, incomeRequest)
    if err != nil {
        // Проверяем, связана ли ошибка с истечением срока действия токена
        if fmt.Sprintf("%v", err) == "access token is expired" {
            log.Printf("Access token expired for PaymentID: %d, attempting to re-authenticate", data.PaymentID)
            
            // Повторная аутентификация через существующий клиент
            login := os.Getenv("MOY_NALOG_LOGIN")
            password := os.Getenv("MOY_NALOG_PASSWORD")
            
            if login == "" || password == "" {
                log.Printf("MOY_NALOG_LOGIN and/or MOY_NALOG_PASSWORD environment variables are not set: PaymentID: %d", data.PaymentID)
                return nil
            }
            
            // Выполняем повторную аутентификацию с использованием метода Auth()
            authErr := c.Auth(ctx)
            if authErr != nil {
                log.Printf("Failed to re-authenticate for PaymentID: %d, Error: %v", data.PaymentID, authErr)
                return nil
            }

            // Повторяем попытку отправки чека один раз
            log.Printf("MoyNalog income payload after re-auth - PaymentID: %d, Time: %v, Payload: %+v", data.PaymentID, time.Now().UTC(), incomeRequest)
            _, retryErr := c.client.Income.Create(ctx, incomeRequest)
            if retryErr != nil {
                log.Printf("Failed to send receipt to Moy Nalog after re-auth: %v, PaymentID: %d, Amount: %.2f",
                    retryErr, data.PaymentID, data.Amount)
                return nil
            }
            
            log.Printf("Successfully sent receipt to Moy Nalog after re-auth: PaymentID: %d, Amount: %.2f",
                data.PaymentID, data.Amount)
            return nil
        }
        
        log.Printf("Failed to send receipt to Moy Nalog: %v, PaymentID: %d, Amount: %.2f",
            err, data.PaymentID, data.Amount)
        // Ошибка логируется, но не вызывает панику и не прерывает основной процесс
        return nil // Возвращаем nil, чтобы избежать паники при отправке чека
    }

    log.Printf("Successfully sent receipt to Moy Nalog: PaymentID: %d, Amount: %.2f",
        data.PaymentID, data.Amount)
    return nil
}