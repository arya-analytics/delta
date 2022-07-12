package fiber

import (
	"github.com/arya-analytics/delta/pkg/sec"
	"github.com/arya-analytics/delta/pkg/sec/token"
	"github.com/cockroachdb/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"strings"
)

const (
	localsUserKey = "userKey"
)

func TokenMiddleware(svc *token.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tk, err := parseToken(c)
		if err != nil {
			return err
		}
		userKey, err := svc.Validate(tk)
		if err != nil {
			return err
		}
		setUserSubject(c, userKey)
		return nil
	}
}

func setUserSubject(c *fiber.Ctx, key uuid.UUID) {
	c.Locals(localsUserKey, sec.NewUserSubject(key))
}

func getUserSubject(c *fiber.Ctx) sec.Subject {
	return c.Locals(localsUserKey).(sec.Subject)
}

type tokenParser func(c *fiber.Ctx) (token string, found bool, err error)

const (
	tokenCookieName               = "token"
	headerTokenPrefix             = "Bearer "
	invalidAuthorizationHeaderMsg = `
	invalid authorization header. Format should be

		'Authorization: Bearer <token>'
	`
)

var tokenParsers = []tokenParser{
	typeParseCookieToken,
	tryParseHeaderToken,
}

func parseToken(c *fiber.Ctx) (string, error) {
	for _, tp := range tokenParsers {
		if tk, found, err := tp(c); found {
			return tk, err
		}
	}
	c.Status(fiber.StatusUnauthorized)
	return "", errors.New("invalid token")
}

func typeParseCookieToken(c *fiber.Ctx) (string, bool, error) {
	tk := c.Cookies(tokenCookieName)
	return tk, len(tk) == 0, nil
}

func tryParseHeaderToken(c *fiber.Ctx) (string, bool, error) {
	authHeader := c.Get("Authorization")
	if len(authHeader) == 0 {
		return "", false, nil
	}
	splitToken := strings.Split(authHeader, headerTokenPrefix)
	if len(splitToken) != 2 {
		return "",
			false,
			errors.New(invalidAuthorizationHeaderMsg)
	}
	return splitToken[1], true, nil
}
