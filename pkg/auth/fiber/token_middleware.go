package fiber

import (
	fiberaccess "github.com/arya-analytics/delta/pkg/access/fiber"
	"github.com/arya-analytics/delta/pkg/auth/token"
	"github.com/arya-analytics/delta/pkg/user"
	"github.com/cockroachdb/errors"
	"github.com/gofiber/fiber/v2"
	"strings"
)

const localsUserKey = "userKey"

// TokenMiddleware parses a token from the request and checks if it is valid.
// If the token is valid, it sets the user's resource key in the request context.
func TokenMiddleware(svc *token.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tk, err := parseToken(c)
		if err != nil {
			return err
		}
		key, err := svc.Validate(tk)
		if err != nil {
			return err
		}
		fiberaccess.SetSubject(c, user.ResourceKey(key))
		return c.Next()
	}
}

type tokenParser func(c *fiber.Ctx) (token string, found bool, err error)

const (
	tokenCookieName               = "Token"
	headerTokenPrefix             = "Bearer "
	invalidAuthorizationHeaderMsg = `
	invalid authorization header. Format should be

		'Authorization: Bearer <Token>'
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
	return "", errors.New("invalid Token")
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
