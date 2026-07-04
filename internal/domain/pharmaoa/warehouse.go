package pharmaoa

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type WarehouseStatus string

const (
	WarehouseStatusEnabled  WarehouseStatus = "enabled"
	WarehouseStatusDisabled WarehouseStatus = "disabled"
)

type WarehouseTemperature struct {
	Controlled bool    `json:"controlled"`
	MinCelsius float64 `json:"minCelsius"`
	MaxCelsius float64 `json:"maxCelsius"`
}

type WarehouseLocation struct {
	ID          string               `json:"id"`
	Code        string               `json:"code"`
	Name        string               `json:"name"`
	Status      WarehouseStatus      `json:"status"`
	Temperature WarehouseTemperature `json:"temperature"`
}

type WarehouseArea struct {
	ID          string               `json:"id"`
	Code        string               `json:"code"`
	Name        string               `json:"name"`
	Status      WarehouseStatus      `json:"status"`
	Temperature WarehouseTemperature `json:"temperature"`
	Locations   []WarehouseLocation  `json:"locations"`
}

type Warehouse struct {
	ID            shared.ID            `json:"id"`
	Code          string               `json:"code"`
	Name          string               `json:"name"`
	Region        string               `json:"region"`
	Status        WarehouseStatus      `json:"status"`
	DisableReason string               `json:"disableReason,omitempty"`
	Temperature   WarehouseTemperature `json:"temperature"`
	Areas         []WarehouseArea      `json:"areas"`
	Meta          shared.AuditMeta     `json:"meta"`
}

type WarehouseInput struct {
	Code        string
	Name        string
	Region      string
	Temperature WarehouseTemperature
	Areas       []WarehouseArea
}

func NewWarehouse(id shared.ID, in WarehouseInput, now time.Time) (*Warehouse, error) {
	if id.IsZero() {
		return nil, fmt.Errorf("id is required")
	}
	if err := validateWarehouseInput(in); err != nil {
		return nil, err
	}
	temperature, err := NormalizeWarehouseTemperature(in.Temperature)
	if err != nil {
		return nil, err
	}
	areas, err := NormalizeWarehouseAreas(in.Areas)
	if err != nil {
		return nil, err
	}
	entity := &Warehouse{
		ID:          id,
		Code:        strings.TrimSpace(in.Code),
		Name:        strings.TrimSpace(in.Name),
		Region:      strings.TrimSpace(in.Region),
		Status:      WarehouseStatusEnabled,
		Temperature: temperature,
		Areas:       areas,
	}
	entity.Meta.Touch(now)
	return entity, nil
}

func (w *Warehouse) Update(in WarehouseInput, now time.Time) error {
	if w == nil {
		return fmt.Errorf("warehouse is required")
	}
	if err := validateWarehouseInput(in); err != nil {
		return err
	}
	temperature, err := NormalizeWarehouseTemperature(in.Temperature)
	if err != nil {
		return err
	}
	areas, err := NormalizeWarehouseAreas(in.Areas)
	if err != nil {
		return err
	}
	w.Code = strings.TrimSpace(in.Code)
	w.Name = strings.TrimSpace(in.Name)
	w.Region = strings.TrimSpace(in.Region)
	w.Temperature = temperature
	w.Areas = areas
	w.Meta.Touch(now)
	return nil
}

func (w *Warehouse) Disable(reason string, now time.Time) error {
	if w == nil {
		return fmt.Errorf("warehouse is required")
	}
	if w.Status == WarehouseStatusDisabled {
		return nil
	}
	w.Status = WarehouseStatusDisabled
	w.DisableReason = strings.TrimSpace(reason)
	w.Meta.Touch(now)
	return nil
}

func (w *Warehouse) ValidateMovementLocation(areaID string, locationID string) error {
	if w == nil {
		return fmt.Errorf("warehouse is required")
	}
	if w.Status == WarehouseStatusDisabled {
		return fmt.Errorf("warehouse is disabled")
	}
	areaID = strings.TrimSpace(areaID)
	locationID = strings.TrimSpace(locationID)
	if areaID == "" {
		return fmt.Errorf("areaId is required")
	}
	if locationID == "" {
		return fmt.Errorf("locationId is required")
	}
	for _, area := range w.Areas {
		if area.ID != areaID && !strings.EqualFold(area.Code, areaID) {
			continue
		}
		if area.Status == WarehouseStatusDisabled {
			return fmt.Errorf("warehouse area is disabled")
		}
		for _, location := range area.Locations {
			if location.ID != locationID && !strings.EqualFold(location.Code, locationID) {
				continue
			}
			if location.Status == WarehouseStatusDisabled {
				return fmt.Errorf("warehouse location is disabled")
			}
			return nil
		}
		return fmt.Errorf("warehouse location not found")
	}
	return fmt.Errorf("warehouse area not found")
}

func NormalizeWarehouseTemperature(in WarehouseTemperature) (WarehouseTemperature, error) {
	if !in.Controlled {
		return WarehouseTemperature{}, nil
	}
	if in.MinCelsius > in.MaxCelsius {
		return WarehouseTemperature{}, fmt.Errorf("temperature minCelsius cannot exceed maxCelsius")
	}
	return in, nil
}

func NormalizeWarehouseAreas(items []WarehouseArea) ([]WarehouseArea, error) {
	out := make([]WarehouseArea, 0, len(items))
	seenIDs := map[string]struct{}{}
	seenCodes := map[string]struct{}{}
	for idx, item := range items {
		item.ID = strings.TrimSpace(item.ID)
		item.Code = strings.TrimSpace(item.Code)
		item.Name = strings.TrimSpace(item.Name)
		if item.ID == "" {
			item.ID = fmt.Sprintf("area-%d", idx+1)
		}
		if item.Code == "" {
			return nil, fmt.Errorf("area code is required")
		}
		if item.Name == "" {
			return nil, fmt.Errorf("area name is required")
		}
		if item.Status == "" {
			item.Status = WarehouseStatusEnabled
		}
		if item.Status != WarehouseStatusEnabled && item.Status != WarehouseStatusDisabled {
			return nil, fmt.Errorf("invalid area status: %s", item.Status)
		}
		if _, ok := seenIDs[item.ID]; ok {
			return nil, fmt.Errorf("duplicate area id: %s", item.ID)
		}
		codeKey := strings.ToLower(item.Code)
		if _, ok := seenCodes[codeKey]; ok {
			return nil, fmt.Errorf("duplicate area code: %s", item.Code)
		}
		temperature, err := NormalizeWarehouseTemperature(item.Temperature)
		if err != nil {
			return nil, err
		}
		locations, err := NormalizeWarehouseLocations(item.Locations)
		if err != nil {
			return nil, err
		}
		item.Temperature = temperature
		item.Locations = locations
		seenIDs[item.ID] = struct{}{}
		seenCodes[codeKey] = struct{}{}
		out = append(out, item)
	}
	return out, nil
}

func NormalizeWarehouseLocations(items []WarehouseLocation) ([]WarehouseLocation, error) {
	out := make([]WarehouseLocation, 0, len(items))
	seenIDs := map[string]struct{}{}
	seenCodes := map[string]struct{}{}
	for idx, item := range items {
		item.ID = strings.TrimSpace(item.ID)
		item.Code = strings.TrimSpace(item.Code)
		item.Name = strings.TrimSpace(item.Name)
		if item.ID == "" {
			item.ID = fmt.Sprintf("location-%d", idx+1)
		}
		if item.Code == "" {
			return nil, fmt.Errorf("location code is required")
		}
		if item.Name == "" {
			return nil, fmt.Errorf("location name is required")
		}
		if item.Status == "" {
			item.Status = WarehouseStatusEnabled
		}
		if item.Status != WarehouseStatusEnabled && item.Status != WarehouseStatusDisabled {
			return nil, fmt.Errorf("invalid location status: %s", item.Status)
		}
		if _, ok := seenIDs[item.ID]; ok {
			return nil, fmt.Errorf("duplicate location id: %s", item.ID)
		}
		codeKey := strings.ToLower(item.Code)
		if _, ok := seenCodes[codeKey]; ok {
			return nil, fmt.Errorf("duplicate location code: %s", item.Code)
		}
		temperature, err := NormalizeWarehouseTemperature(item.Temperature)
		if err != nil {
			return nil, err
		}
		item.Temperature = temperature
		seenIDs[item.ID] = struct{}{}
		seenCodes[codeKey] = struct{}{}
		out = append(out, item)
	}
	return out, nil
}

func validateWarehouseInput(in WarehouseInput) error {
	for field, value := range map[string]string{
		"code":   in.Code,
		"name":   in.Name,
		"region": in.Region,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	if len(in.Areas) == 0 {
		return fmt.Errorf("at least one area is required")
	}
	return nil
}
