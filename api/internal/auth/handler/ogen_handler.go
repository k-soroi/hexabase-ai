package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/hexabase/hexabase-ai/api/internal/auth/domain"
	"github.com/hexabase/hexabase-ai/api/internal/shared/infrastructure/server/ogen"
)

// OgenAuthHandler implements ogen.Handler interface for auth endpoints
type OgenAuthHandler struct {
	authService domain.Service
	logger      *slog.Logger
}

// NewOgenAuthHandler creates a new ogen auth handler
func NewOgenAuthHandler(authService domain.Service, logger *slog.Logger) *OgenAuthHandler {
	return &OgenAuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// StartAuthSignUp implements ogen.Handler
func (h *OgenAuthHandler) StartAuthSignUp(
	ctx context.Context,
	req ogen.OptSignUpRequest,
	params ogen.StartAuthSignUpParams,
) (ogen.StartAuthSignUpRes, error) {
	codeChallenge := ""
	codeChallengeMethod := ""

	if req.Set {
		if cc, ok := req.Value.CodeChallenge.Get(); ok {
			codeChallenge = cc
		}

		if cm, ok := req.Value.CodeChallengeMethod.Get(); ok {
			codeChallengeMethod = cm
		}
	}
	// Convert ogen request to domain request
	domainReq := &domain.SignUpAuthRequest{
		Provider:            string(params.Provider),
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
	}

	// Call service
	authURL, state, err := h.authService.GetAuthURLForSignUp(ctx, domainReq)
	if err != nil {
		h.logger.Error("failed to generate auth URL for sign-up",
			"error", err,
			"provider", params.Provider)

		return h.NewError(ctx, err), nil
	}

	// Parse URL for ogen
	parsedURL, err := url.Parse(authURL)
	if err != nil {
		h.logger.Error("failed to parse auth URL",
			"error", err,
			"url", authURL)

		return h.NewError(ctx, err), nil
	}

	// Return response
	resp := &ogen.SignUpResponse{
		Provider: ogen.NewOptString(string(params.Provider)),
		AuthURL:  ogen.NewOptURI(*parsedURL),
		State:    ogen.NewOptString(state),
	}

	return resp, nil
}

// NewError implements ogen.Handler
func (h *OgenAuthHandler) NewError(ctx context.Context, err error) *ogen.SignUpErrorResponseStatusCode {
	// Default to internal server error
	statusCode := http.StatusInternalServerError
	message := "authentication service error"

	// Map specific errors to appropriate status codes and messages
	switch {
	case errors.Is(err, domain.ErrUnsupportedProvider):
		statusCode = http.StatusBadRequest
		message = "unsupported authentication provider"
	case errors.Is(err, domain.ErrInvalidRequest):
		statusCode = http.StatusBadRequest
		message = "invalid request parameters"
	case err == nil:
		// This should never happen in normal ogen flow as NewError is only called
		// when err != nil. If we reach here, it indicates a programming error.
		h.logger.Warn("NewError called with nil error - this indicates a bug")
	default:
		// For any other error, log the details but don't expose them
		h.logger.Error("authentication error",
			"error", err,
			"type", fmt.Sprintf("%T", err))
	}

	return &ogen.SignUpErrorResponseStatusCode{
		StatusCode: statusCode,
		Response: ogen.SignUpErrorResponse{
			Error: ogen.NewOptString(message),
		},
	}
}
