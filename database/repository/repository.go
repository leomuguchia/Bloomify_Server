package repository

import (
	providerRepo "bloomify/database/repository/provider"
	schedulerRepo "bloomify/database/repository/scheduler"
	timeslotRepo "bloomify/database/repository/timeslot"
	userRepo "bloomify/database/repository/user"
)

// Re-export the ProviderRepository interface and constructors.
type ProviderRepository = providerRepo.ProviderRepository

type ProviderSearchCriteria = providerRepo.ProviderSearchCriteria

var NewMongoProviderRepo = providerRepo.NewMongoProviderRepo

// Re-export the UserRepository interface and constructor.
type UserRepository = userRepo.UserRepository

var NewMongoUserRepository = userRepo.NewMongoUserRepo

// Re-export the SchedulerRepository interface and constructor.
type SchedulerRepository = schedulerRepo.SchedulerRepository

var NewMongoSchedulerRepo = schedulerRepo.NewMongoSchedulerRepo

// Re-export the TimeslotsRepository interface and constructor.
type TimeslotsRepository = timeslotRepo.TimeSlotRepository

var NewMongoTimeSlotRepo = timeslotRepo.NewMongoTimeSlotRepo
