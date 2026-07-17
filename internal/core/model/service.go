package model

type Service struct {
	repo Repository

	registry Registry

	validator Validator
}
