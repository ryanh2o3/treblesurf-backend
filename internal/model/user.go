package model

import "time"

type User struct {
	UUID       string `json:"uuid" dynamodbav:"uuid"`
	Email      string `json:"email" dynamodbav:"email"`
	Name       string `json:"name" dynamodbav:"name"`
	Picture    string `json:"picture" dynamodbav:"picture"`
	FamilyName string `json:"family_name" dynamodbav:"family_name"`
	GivenName  string `json:"given_name" dynamodbav:"given_name"`
	CreatedAt  string `json:"created_at" dynamodbav:"created_at"`
	LastLogin  string `json:"last_login" dynamodbav:"last_login"`
	Theme      string `json:"theme" dynamodbav:"theme"`
	Role       string `json:"role,omitempty" dynamodbav:"role,omitempty"`
}

type UserSession struct {
	Email       string    `json:"email"`
	SessionID   string    `json:"session_id"`
	UserAgent   string    `json:"user_agent"`
	IPAddress   string    `json:"ip_address"`
	CreatedAt   time.Time `json:"created_at"`
	LastActive  time.Time `json:"last_active"`
	CSRFToken   string    `json:"csrf_token"`
}

type TokenRequest struct {
	IDToken string `json:"id_token"`
}
