package copy

import (
	"doubleboiler/config"
	"fmt"
)

func VerificationEmail(verificationUrl string, orgName string) (html, text string) {
	html = fmt.Sprintf(`
	Hi there! %s is using %s.
	<br><br>
	We've got your account all set up and ready to go. All that's left is to confirm your email address.
	<br><br>
	Just click <a href="%s">here</a> and you're all set!
	<br><br>
	If there's a problem with that link, paste this URL into a browser:
	<br><br>
	%s
	<br><br>
	Cheers,
	<br><br>
	The team at %s
	`, orgName, config.URI, verificationUrl, verificationUrl, config.NAME)

	text = fmt.Sprintf(`
Hi there! %s is using %s.

We've got your account all set up and ready to go. All that's left is to confirm your email address.

Just visit this URL:

%s

Cheers,

The team at %s
	`, orgName, config.URI, verificationUrl, config.NAME)

	return
}

func OrgAdditionEmail(organisationName string) (html, text string) {
	html = fmt.Sprintf(`
	Hi there! You've been added to the %s organisation on %s
	<br><br>
	It'll all be there waiting for you next time you log in.
	<br><br>
	Cheers,
	<br><br>
	The team at %s
	`, organisationName, config.URI, config.NAME)

	text = fmt.Sprintf(`
Hi there! You've been added to the %s organisation on %s

It'll all be there waiting for you next time you log in.

Cheers,

The team at %s
	`, organisationName, config.URI, config.NAME)

	return
}

func OrgInviteEmail(organisationName, verificationUrl string) (html, text string) {
	html = fmt.Sprintf(`
	Hi there! You've been invited to join the %s organisation on %s
	<br><br>
	We've got your account all set up and ready to go. All that's left is to confirm your email address.
	<br><br>
	Just click <a href="%s">here</a> and you're all set!
	<br><br>
	If there's a problem with that link, paste this URL into a browser:
	<br><br>
	%s
	<br><br>
	Cheers,
	<br><br>
	The team at %s
	`, organisationName, config.URI, verificationUrl, verificationUrl, config.NAME)

	text = fmt.Sprintf(`
Hi there! You've been invited to join the %s organisation on %s

We've got your account all set up and ready to go. All that's left is to confirm your email address.

Just visit this URL:

%s

Cheers,

The team at %s
	`, organisationName, config.URI, verificationUrl, config.NAME)

	return
}

func PasswordResetEmail(resetUrl string) (html, text string) {
	html = fmt.Sprintf(`
	Hi there! A password reset has been requested for your <a href="%s">%s</a> account.
	<br><br>
	<br><br>
	To set a new password, click <a href="%s">here</a> and you're all set!
	<br><br>
	If there's a problem with that link, paste this URL into a browser:
	<br><br>
	%s
	<br><br>
	Cheers,
	<br><br>
	The team at %s
	`, config.URI, config.NAME, resetUrl, resetUrl, config.NAME)

	text = fmt.Sprintf(`
Hi there! A password reset has been requested for your account at %s

To set a new password, visit this URL:

%s

Cheers,

The team at %s
	`, config.NAME, resetUrl, config.NAME)

	return
}

func EmailChangedEmail(target, old string) (html, text string) {
	html = fmt.Sprintf(`
	Hi there! Your email address for your %s account has been changed from %s to %s.
	<br><br>
	If this is not something you expected to happen, please reply to this email and let us know immediately.
	<br><br>
	Thanks,
	<br><br>
	The team at %s
	`, config.NAME, old, target, config.NAME)

	text = fmt.Sprintf(`
Hi there! Your email address for your %s account has been changed from %s to %s.

If this is not something you expected to happen, please reply to this email and let us know immediately.

Thanks,

The team at %s
	`, config.NAME, old, target, config.NAME)

	return
}
