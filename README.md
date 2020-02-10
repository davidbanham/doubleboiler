## Double Boiler

### Quickstart
Edit the brand field in the Makefile to the name of your app
Edit the email address in seed/seed_user.up.sql to your own
Set a new secret, hash and block key in .envrc_example. The hash and block keys must be the same number of characters

I like to manage my environment variables with [direnv](https://direnv.net/)

You'll need AWS credentials available through the environment in order to send emails via SES

To generate new logos you'll need inkscape, scour, and convert installed

```
# make rename
# cp .envrc_example .envrc
# direnv allow
# make .db_init
# make logos_to_paths
# make rummage < seed/seed_user.up.sql
# make live_reload
```

Now login with your seeded admin email address and the password `notasecret`

When you're ready to start adding to your app:

```
# make new_resource
```

### Deploying to Cloud Run
Fill out the [PROJECT_ID] and [REGION] placeholders in cloudbuild.yaml as per the instructions [here](https://cloud.google.com/cloud-build/docs/deploying-builds/deploy-cloud-run)

Then:

```
# make deploy
```
