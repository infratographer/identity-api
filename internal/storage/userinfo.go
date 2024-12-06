package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"go.infratographer.com/identity-api/internal/crdbx"
	"go.infratographer.com/identity-api/internal/types"
	"go.infratographer.com/x/gidx"
)

const (
	jwtClaimSubject = "sub"
	jwtClaimName    = "name"
	jwtClaimEmail   = "email"
	jwtClaimIssuer  = "iss"
)

var (
	errMissingClaim = errors.New("missing required claim")
	errInvalidClaim = errors.New("invalid claim value")
)

var userInfoCols = struct {
	ID       string
	Name     string
	Email    string
	Subject  string
	IssuerID string
}{
	ID:       "id",
	Name:     "name",
	Email:    "email",
	Subject:  "sub",
	IssuerID: "iss_id",
}

func generateSubjectID(prefix, iss, sub string) (gidx.PrefixedID, error) {
	// Concatenate the iss and sub values, then hash them
	issSub := iss + sub
	issSubHash := sha256.Sum256([]byte(issSub))

	digest := base64.RawURLEncoding.EncodeToString(issSubHash[:])

	// Concatenate the prefix with the digest
	out := prefix + "-" + digest

	return gidx.Parse(out)
}

type userInfoService struct {
	db *sql.DB
}

type userInfoServiceOpt func(*userInfoService)

func newUserInfoService(db *sql.DB, opts ...userInfoServiceOpt) (*userInfoService, error) {
	s := &userInfoService{
		db: db,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

// LookupUserInfoByClaims fetches UserInfo from the store.
// This does not make an HTTP call with the subject token, so for this
// data to be available, the data must have already be fetched and
// stored.
func (s userInfoService) LookupUserInfoByClaims(ctx context.Context, iss, sub string) (types.UserInfo, error) {
	selectCols := withQualifier([]string{
		userInfoCols.Name,
		userInfoCols.Email,
		userInfoCols.Subject,
	}, "ui")

	selectCols = append(selectCols, "i."+issuerCols.URI)

	selects := strings.Join(selectCols, ",")

	stmt := fmt.Sprintf(`
	SELECT %[1]s FROM user_info ui
        JOIN issuers i ON ui.%[2]s = i.%[3]s
        WHERE i.%[4]s = $1 and ui.%[5]s = $2`,
		selects,
		userInfoCols.IssuerID,
		issuerCols.ID,
		issuerCols.URI,
		userInfoCols.Subject,
	)

	var row *sql.Row

	tx, err := getContextTx(ctx)

	switch err {
	case nil:
		row = tx.QueryRowContext(ctx, stmt, iss, sub)
	case ErrorMissingContextTx:
		row = s.db.QueryRowContext(ctx, stmt, iss, sub)
	default:
		return types.UserInfo{}, err
	}

	var ui types.UserInfo

	err = row.Scan(&ui.Name, &ui.Email, &ui.Subject, &ui.Issuer)

	if errors.Is(err, sql.ErrNoRows) {
		return types.UserInfo{}, types.ErrUserInfoNotFound
	}

	return ui, err
}

func (s userInfoService) LookupUserInfoByID(ctx context.Context, id gidx.PrefixedID) (types.UserInfo, error) {
	selectCols := withQualifier([]string{
		userInfoCols.ID,
		userInfoCols.Name,
		userInfoCols.Email,
		userInfoCols.Subject,
	}, "ui")

	selectCols = append(selectCols, "i."+issuerCols.URI)

	selects := strings.Join(selectCols, ",")

	stmt := fmt.Sprintf(`
        SELECT %[1]s FROM user_info ui
        JOIN issuers i ON ui.%[2]s = i.%[3]s
        WHERE ui.%[4]s = $1
        `, selects, userInfoCols.IssuerID, issuerCols.ID, userInfoCols.ID)

	var row *sql.Row

	tx, err := getContextTx(ctx)

	switch err {
	case nil:
		row = tx.QueryRowContext(ctx, stmt, id)
	case ErrorMissingContextTx:
		row = s.db.QueryRowContext(ctx, stmt, id)
	default:
		return types.UserInfo{}, err
	}

	var ui types.UserInfo

	err = row.Scan(&ui.ID, &ui.Name, &ui.Email, &ui.Subject, &ui.Issuer)

	if errors.Is(err, sql.ErrNoRows) {
		return types.UserInfo{}, types.ErrUserInfoNotFound
	}

	return ui, err
}

// LookupUserOwnerID finds the Owner ID of the Issuer for the given User ID.
func (s userInfoService) LookupUserOwnerID(ctx context.Context, id gidx.PrefixedID) (gidx.PrefixedID, error) {
	stmt := `
        SELECT issuers.owner_id
		FROM issuers, user_info
		WHERE
			issuers.id = user_info.iss_id AND
			user_info.id = $1
    `

	var row *sql.Row

	tx, err := getContextTx(ctx)

	switch err {
	case nil:
		row = tx.QueryRowContext(ctx, stmt, id)
	case ErrorMissingContextTx:
		row = s.db.QueryRowContext(ctx, stmt, id)
	default:
		return gidx.NullPrefixedID, err
	}

	var ownerID gidx.PrefixedID

	err = row.Scan(&ownerID)

	if errors.Is(err, sql.ErrNoRows) {
		return gidx.NullPrefixedID, types.ErrUserInfoNotFound
	}

	return ownerID, err
}

// LookupUserInfosByIssuerID lists users for an issuer.
func (s *userInfoService) LookupUserInfosByIssuerID(ctx context.Context, id gidx.PrefixedID, pagination crdbx.Paginator) (types.UserInfos, error) {
	paginate := crdbx.Paginate(pagination, crdbx.ContextAsOfSystemTime(ctx, "-1m")).WithQualifier("user_info")

	selectCols := withQualifier([]string{
		userInfoCols.ID,
		userInfoCols.Name,
		userInfoCols.Email,
		userInfoCols.Subject,
	}, "user_info")

	selectCols = append(selectCols, "issuers."+issuerCols.URI)

	selects := strings.Join(selectCols, ",")

	query := fmt.Sprintf(`
			SELECT %[1]s
			FROM user_info, issuers
			%[4]s
			WHERE issuers.%[3]s = $1 AND user_info.iss_id = issuers.id %[5]s %[6]s %[7]s
        `, selects, userInfoCols.IssuerID, issuerCols.ID,
		paginate.AsOfSystemTime(),
		paginate.AndWhere(2), //nolint:mnd
		paginate.OrderClause(),
		paginate.LimitClause(),
	)

	rows, err := s.db.QueryContext(ctx, query, paginate.Values(id)...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var users types.UserInfos

	for rows.Next() {
		var model types.UserInfo

		err = rows.Scan(&model.ID, &model.Name, &model.Email, &model.Subject, &model.Issuer)
		if err != nil {
			return nil, err
		}

		users = append(users, model)
	}

	return users, nil
}

// StoreUserInfo is used to store user information by issuer and
// subject pairs. UserInfo is unique to issuer/subject pairs.
func (s userInfoService) StoreUserInfo(ctx context.Context, userInfo types.UserInfo) (types.UserInfo, error) {
	if len(userInfo.Issuer) == 0 {
		return types.UserInfo{}, fmt.Errorf("%w: issuer is empty", types.ErrInvalidUserInfo)
	}

	if len(userInfo.Subject) == 0 {
		return types.UserInfo{}, fmt.Errorf("%w: subject is empty", types.ErrInvalidUserInfo)
	}

	tx, err := getContextTx(ctx)
	if err != nil {
		return types.UserInfo{}, err
	}

	row := tx.QueryRowContext(ctx, `
        SELECT id FROM issuers WHERE uri = $1
        `, userInfo.Issuer)

	var issuerID gidx.PrefixedID

	err = row.Scan(&issuerID)
	switch err {
	case nil:
	case sql.ErrNoRows:
		return types.UserInfo{}, types.ErrorIssuerNotFound
	default:
		return types.UserInfo{}, err
	}

	insertCols := strings.Join([]string{
		userInfoCols.ID,
		userInfoCols.Name,
		userInfoCols.Email,
		userInfoCols.Subject,
		userInfoCols.IssuerID,
	}, ",")

	var newID gidx.PrefixedID

	if userInfo.ID.String() == "" {
		newID, err = generateSubjectID(types.IdentityUserIDPrefix, userInfo.Issuer, userInfo.Subject)
		if err != nil {
			return types.UserInfo{}, err
		}
	} else {
		newID = userInfo.ID
	}

	q := fmt.Sprintf(`INSERT INTO user_info (%[1]s) VALUES (
            $1, $2, $3, $4, $5
	) ON CONFLICT (%[2]s, %[3]s)
        DO UPDATE SET %[2]s = excluded.%[2]s, %[3]s = excluded.%[3]s
        RETURNING id`,
		insertCols,
		userInfoCols.Subject,
		userInfoCols.IssuerID,
	)

	row = tx.QueryRowContext(ctx, q,
		newID, userInfo.Name, userInfo.Email, userInfo.Subject, issuerID,
	)

	var userID gidx.PrefixedID

	err = row.Scan(&userID)
	if err != nil {
		return types.UserInfo{}, err
	}

	userInfo.ID = userID

	return userInfo, err
}

func parseClaim(claims map[string]any, key string, required bool) (string, error) {
	rawVal, ok := claims[key]
	if !ok {
		rawVal = ""
	}

	val, ok := rawVal.(string)
	if !ok {
		return "", errInvalidClaim
	}

	if required && val == "" {
		return "", errMissingClaim
	}

	return val, nil
}

func (s userInfoService) ParseUserInfoFromClaims(claims map[string]any) (types.UserInfo, error) {
	iss, err := parseClaim(claims, jwtClaimIssuer, true)
	if err != nil {
		return types.UserInfo{}, err
	}

	sub, err := parseClaim(claims, jwtClaimSubject, true)
	if err != nil {
		return types.UserInfo{}, err
	}

	name, err := parseClaim(claims, jwtClaimName, false)
	if err != nil {
		return types.UserInfo{}, err
	}

	email, err := parseClaim(claims, jwtClaimEmail, false)
	if err != nil {
		return types.UserInfo{}, err
	}

	out := types.UserInfo{
		Issuer:  iss,
		Subject: sub,
		Name:    name,
		Email:   email,
	}

	return out, nil
}
