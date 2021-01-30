# Pretty Profile Readme

Generate README.md with GitHub and WakaTime data.

## Prepare

1. Create a profile repository.
This repository will show in your profile. The repository name must be your github name. For example, my repository name is `gccio`.

2. Update the markdown file with two comments
```text
<!--START_SECTION:waka-->
<!--END_SECTION:waka-->
```

3. Get WakaTime API Key. If you dont want to use wakatime, you can ignore this step.

   Go to <https://wakatime.com> and create an account.

   Get your WakaTime API Key from your [Account Settings in WakaTime](https://wakatime.com/settings/account).

4. Get a GitHub Token from [here](https://github.com/settings/tokens)
The token need `repo` and `user` scope access.

## How to use it

1. Set secret in repository settings.

   `GH_TOKEN=<your github access token>`

   `WAKATIME_API_KEY=<your wakatime api key>`(if you have)

2. Add github workflows.
```yml
name: Waka Readme

on:
  schedule:
    - cron: '0 0 * * *'
  workflow_dispatch:
jobs:
  update-readme:
    name: Update Readme
    runs-on: ubuntu-20.04
    steps:
      - uses: gccio/pretty-profile-readme@v1.0.0
        env:
          WAKATIME_API_KEY: ${{ secrets.WAKATIME_API_KEY }}
          GH_TOKEN: ${{ secrets.GH_TOKEN }}
          TIMEZONE: "Asia/Shanghai"
```
