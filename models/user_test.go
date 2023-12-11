package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
)

func init() {
	modelsUnderTest = append(modelsUnderTest, userFix())
	modelCollectionsUnderTest = append(modelCollectionsUnderTest, usersFix())
}

func userFixture() (user User) {
	email := fmt.Sprintf("%s@example.com", randString())
	user.New(email, randString())
	return
}

func (User) blank() model {
	return &User{}
}

func (i User) id() string {
	return i.ID
}

func (i *User) nullDynamicValues() {
	i.CreatedAt = time.Time{}
	i.UpdatedAt = time.Time{}
	i.Revision = ""
	i.recoveryCodes = NullStringList{}
}

func (User) tablename() string {
	return "users"
}

func (Users) tablename() string {
	return "users"
}

func (Users) blank() models {
	return &Users{}
}

func usersFix() modelCollectionFixture {
	return modelCollectionFixture{
		deps: []model{},
		collection: &Users{
			Data: []User{
				userFixture(),
				userFixture(),
			},
		},
	}
}

func userFix() []model {
	fix := userFixture()
	return []model{
		&fix,
	}
}

func (this Users) data() []model {
	ret := []model{}
	for _, m := range this.Data {
		ret = append(ret, &m)
	}
	return ret
}

func TestUserRevisionCollision(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	fix := userFixture()
	assert.Nil(t, fix.Save(ctx))
	fix.Revision = "yeahnah"
	assert.Error(t, fix.Save(ctx))
	assert.Equal(t, ErrWrongRev, fix.Save(ctx))

	closeTx(t, ctx)
}

func TestUserRevisionChange(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	fix := userFixture()
	assert.Nil(t, fix.Save(ctx))
	firstRev := fix.Revision
	assert.Nil(t, fix.Save(ctx))
	assert.NotEqual(t, firstRev, fix.Revision)

	closeTx(t, ctx)
}

func TestUserGenerate2FA(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)
	defer closeTx(t, ctx)

	fix := userFixture()
	assert.Nil(t, fix.Save(ctx))

	key, err := fix.Generate2FA(ctx, "", "")
	assert.Nil(t, err)
	assert.NotNil(t, key)
}

func TestUserValidate2FA(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)
	defer closeTx(t, ctx)

	fix := userFixture()
	assert.Nil(t, fix.Save(ctx))

	discard, err := fix.Generate2FA(ctx, "", "")
	assert.Nil(t, err)
	assert.NotNil(t, discard)

	// It should allow key generation without validation before first activation
	key, err := fix.Generate2FA(ctx, "", "")
	assert.Nil(t, err)
	assert.NotNil(t, key)

	// It shouldn't resist brute force before first activation
	for i := 0; i < 10; i++ {
		badCode, err := totp.GenerateCode(key.Secret(), time.Now().Add(-5*time.Minute))
		notOkay, err := fix.Validate2FA(ctx, badCode, "")
		assert.Nil(t, err)
		assert.False(t, notOkay)
	}

	assert.Equal(t, 0, len(fix.recoveryCodes.Strings))
	assert.False(t, fix.recoveryCodes.Valid)

	code, err := totp.GenerateCode(key.Secret(), time.Now())
	assert.Nil(t, err)

	okay, err := fix.Validate2FA(ctx, code, "")
	assert.Nil(t, err)
	assert.True(t, okay)

	codes, err := fix.generateRecoveryCodes(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 10, len(codes))

	assert.Equal(t, 10, len(fix.recoveryCodes.Strings))
	assert.True(t, fix.recoveryCodes.Valid)

	// It should deny key generation without validation after first activation
	nah, err := fix.Generate2FA(ctx, "", "")
	assert.NotNil(t, err)
	assert.Nil(t, nah)
	// Reset brute force counter
	db := ctx.Value("tx").(Querier)
	_, err = db.ExecContext(ctx, "UPDATE users SET totp_failure_count = 0, totp_last_failure = $2 WHERE id = $1", fix.ID, time.Now().Add(-5*time.Minute))
	assert.Nil(t, err)

	badCode, err := totp.GenerateCode(key.Secret(), time.Now().Add(-5*time.Minute))
	assert.Nil(t, err)

	for i := 0; i < 4; i++ {
		notOkay, err := fix.Validate2FA(ctx, badCode, "")
		assert.Nil(t, err)
		assert.False(t, notOkay)
	}

	tooManyFailures, err := fix.Validate2FA(ctx, badCode, "")
	assert.NotNil(t, err)
	assert.False(t, tooManyFailures)

	_, err = db.ExecContext(ctx, "UPDATE users SET totp_last_failure = $2 WHERE id = $1", fix.ID, time.Now().Add(-5*time.Minute))
	assert.Nil(t, err)

	waitedABit, err := fix.Validate2FA(ctx, badCode, "")
	assert.Nil(t, err)
	assert.False(t, waitedABit)

	failedAgain, err := fix.Validate2FA(ctx, badCode, "")
	assert.NotNil(t, err)
	assert.False(t, failedAgain)

	db = ctx.Value("tx").(Querier)
	_, err = db.ExecContext(ctx, "UPDATE users SET totp_last_failure = $2 WHERE id = $1", fix.ID, time.Now().Add(-5*time.Minute))
	assert.Nil(t, err)

	gotItRight, err := fix.Validate2FA(ctx, code, "")
	assert.Nil(t, err)
	assert.True(t, gotItRight)

	for i := 0; i < 4; i++ {
		notOkay, err := fix.Validate2FA(ctx, badCode, "")
		assert.Nil(t, err)
		assert.False(t, notOkay)
	}

	getItTogether, err := fix.Validate2FA(ctx, badCode, "")
	assert.NotNil(t, err)
	assert.False(t, getItTogether)

	noRecoveryEither, err := fix.Validate2FA(ctx, "", fix.recoveryCodes.Strings[0])
	assert.NotNil(t, err)
	assert.False(t, noRecoveryEither)

	db = ctx.Value("tx").(Querier)
	_, err = db.ExecContext(ctx, "UPDATE users SET totp_last_failure = $2 WHERE id = $1", fix.ID, time.Now().Add(-5*time.Minute))
	assert.Nil(t, err)

	firstRecoveryCode := fix.recoveryCodes.Strings[0]

	recoveryOkayNow, err := fix.Validate2FA(ctx, "", firstRecoveryCode)
	assert.Nil(t, err)
	assert.True(t, recoveryOkayNow)

	assert.Equal(t, 9, len(fix.recoveryCodes.Strings))
	assert.True(t, fix.recoveryCodes.Valid)

	noCodeReuse, err := fix.Validate2FA(ctx, "", firstRecoveryCode)
	assert.Nil(t, err)
	assert.False(t, noCodeReuse)

	recoveryMultipleTimes, err := fix.Validate2FA(ctx, "", fix.recoveryCodes.Strings[0])
	assert.Nil(t, err)
	assert.True(t, recoveryMultipleTimes)

	assert.Equal(t, 8, len(fix.recoveryCodes.Strings))
	assert.True(t, fix.recoveryCodes.Valid)
}
