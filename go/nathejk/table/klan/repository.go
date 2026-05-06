package klan

import (
	"nathejk.dk/nathejk/table/signup"
)

type repository struct {
	Signup             signup.Queries
	TotalMemberCount   uint32
	TeamMinMemberCount uint32
	TeamMaxMemberCount uint32
}

type external func(*repository)

func WithSignup(q signup.Queries) external {
	return func(r *repository) {
		r.Signup = q
	}
}

func WithTeamMinMemberCount(val uint32) external {
	return func(r *repository) {
		r.TeamMinMemberCount = val
	}
}

func WithTeamMaxMemberCount(val uint32) external {
	return func(r *repository) {
		r.TeamMaxMemberCount = val
	}
}

func WithTotalMemberCount(val uint32) external {
	return func(r *repository) {
		r.TotalMemberCount = val
	}
}

func NewRepository(es ...external) repository {
	r := repository{}
	for _, with := range es {
		with(&r)
	}
	return r
}
