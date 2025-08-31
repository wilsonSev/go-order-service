package codec

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/wilsonSev/go-order-service/internal/model"
)

var ErrInvalidJSON = errors.New("invalid json")
var ErrInvalidOrder = errors.New("invalid order")

func ParseAndValidate(raw []byte) (*model.Order, error) {
	var o model.Order
	if err := json.Unmarshal(raw, &o); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}

	// Базовые проверки
	if o.OrderUID == "" {
		return nil, fmt.Errorf("%w: missing order_uid", ErrInvalidOrder)
	}
	if o.TrackNumber == "" {
		return nil, fmt.Errorf("%w: missing track_number", ErrInvalidOrder)
	}
	if len(o.Items) == 0 {
		return nil, fmt.Errorf("%w: no items", ErrInvalidOrder)
	}

	return &o, nil
}
