package services

import (
	"errors"
	"math"

	"go-fiber/internal/models"
	"go-fiber/internal/repositories"
	"go-fiber/internal/utils"

	"github.com/google/uuid"
)

type ShippingService struct {
	configRepo *repositories.ShippingConfigRepository
	shopRepo   *repositories.ShopRepository
}

func NewShippingService(
	configRepo *repositories.ShippingConfigRepository,
	shopRepo *repositories.ShopRepository,
) *ShippingService {
	return &ShippingService{
		configRepo: configRepo,
		shopRepo:   shopRepo,
	}
}

type ShippingEstimateInput struct {
	ShopID            string  `json:"shop_id" validate:"required"`
	ShippingLatitude  float64 `json:"shipping_latitude" validate:"required,min=-90,max=90"`
	ShippingLongitude float64 `json:"shipping_longitude" validate:"required,min=-180,max=180"`
}

type ShippingEstimateResult struct {
	DistanceKm    float64 `json:"distance_km"`
	BaseFee       float64 `json:"base_fee"`
	KmBasedFee    float64 `json:"km_based_fee"`
	TotalFee      float64 `json:"total_fee"`
	MaxDistanceKm float64 `json:"max_distance_km"`
}

func (s *ShippingService) Calculate(shopID uuid.UUID, shippingLat, shippingLong float64) (*ShippingEstimateResult, error) {
	config, err := s.configRepo.Get()
	if err != nil {
		return nil, errors.New("SHIPPING_CONFIG_NOT_FOUND")
	}

	shop, err := s.shopRepo.FindByID(shopID)
	if err != nil {
		return nil, errors.New("shop not found")
	}

	if shop.Latitude == 0 && shop.Longitude == 0 {
		return nil, errors.New("SHOP_LOCATION_NOT_SET")
	}

	distance := utils.HaversineDistance(shop.Latitude, shop.Longitude, shippingLat, shippingLong)
	distance = math.Round(distance*100) / 100

	if distance > config.MaxDistanceKm {
		return nil, errors.New("OUTSIDE_DELIVERY_RANGE")
	}

	baseFee := config.BaseFee
	kmBasedFee := config.PerKmRate * distance
	rawFee := baseFee + kmBasedFee
	totalFee := utils.CeilToNearest(rawFee, 1000)

	return &ShippingEstimateResult{
		DistanceKm:    distance,
		BaseFee:       baseFee,
		KmBasedFee:    kmBasedFee,
		TotalFee:      totalFee,
		MaxDistanceKm: config.MaxDistanceKm,
	}, nil
}

func (s *ShippingService) GetConfig() (*models.ShippingConfig, error) {
	return s.configRepo.Get()
}

type UpdateShippingConfigInput struct {
	BaseFee       float64 `json:"base_fee" validate:"required,min=0"`
	PerKmRate     float64 `json:"per_km_rate" validate:"required,min=0"`
	MaxDistanceKm float64 `json:"max_distance_km" validate:"required,gt=0"`
}

func (s *ShippingService) UpdateConfig(input UpdateShippingConfigInput) (*models.ShippingConfig, error) {
	config, err := s.configRepo.Get()
	if err != nil {
		return nil, errors.New("SHIPPING_CONFIG_NOT_FOUND")
	}

	config.BaseFee = input.BaseFee
	config.PerKmRate = input.PerKmRate
	config.MaxDistanceKm = input.MaxDistanceKm

	if err := s.configRepo.Update(config); err != nil {
		return nil, err
	}
	return config, nil
}
