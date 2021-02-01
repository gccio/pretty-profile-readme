package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type user struct {
	GHToken              string
	WakaTimeAPIKey       string
	UserInfo             *UserInfo
	UserData             *UserData
	WakaTimeStats        *WakaTimeStats
	RepositoryCommitInfo []*RepositoryCommitInfo
}

func Init(GHToken, WakaTimeAPIKey string) *user {
	u := &user{
		GHToken:              GHToken,
		WakaTimeAPIKey:       WakaTimeAPIKey,
		UserInfo:             &UserInfo{},
		UserData:             &UserData{},
		WakaTimeStats:        &WakaTimeStats{},
		RepositoryCommitInfo: []*RepositoryCommitInfo{},
	}
	if err := u.sendRequestToGithub(u.UserInfo, nil); err != nil {
		panic(fmt.Sprintf("get user info failed with error: %v", err))
	}

	if err := u.sendRequestToGithub(u.UserData, map[string]interface{}{
		"login": githubv4.String(u.UserInfo.Viewer.Login),
		"from":  githubv4.DateTime{Time: time.Date(time.Now().Year(), 1, 1, 00, 00, 00, 00, time.UTC)},
	}); err != nil {
		panic(fmt.Sprintf("get user data failed with error: %v", err))
	}

	if u.WakaTimeAPIKey != "" {
		if err := u.getWakaTimeStats(); err != nil {
			panic(fmt.Sprintf("get wakatime stats failed with error:%v", err))
		}
	}

	return u
}

// UserInfo get userinfo.  and id.
type UserInfo struct {
	Viewer struct {
		Login githubv4.String
		ID    githubv4.ID
	}
}

type RepositoryConnection struct {
	TotalCount     githubv4.Int
	TotalDiskUsage githubv4.Int
	Edges          []struct {
		Node struct {
			Owner struct {
				Login githubv4.String
			}
			DiskUsage        githubv4.Int
			Name             githubv4.String
			NameWithOwner    githubv4.String
			IsPrivate        githubv4.Boolean
			DefaultBranchRef struct {
				Name githubv4.String
			}
			PrimaryLanguage struct {
				Color githubv4.String
				Name  githubv4.String
				ID    githubv4.ID
			}
		}
	}
}

type IssueComments struct {
	Edges []struct {
		Node struct {
			BodyHTML githubv4.HTML `graphql:"bodyHTML"`
			URL      githubv4.URI
			Issue    struct {
				Title githubv4.String
			}
		}
	}
}

type Issues struct {
	Edges []struct {
		Node struct {
			URL       githubv4.URI
			Title     githubv4.String
			UpdatedAt githubv4.DateTime
			State     githubv4.IssueState
		}
	}
}

type RepositoryCommitInfo struct {
	Repository struct {
		Name githubv4.String
		Ref  struct {
			Target struct {
				Commit struct {
					History struct {
						Edges []struct {
							Node struct {
								CommittedDate githubv4.Date
							}
						}
					} `graphql:"history(first: $last, author:{id: $id}, since: $since)"`
				} `graphql:"... on Commit"`
			}
		} `graphql:"ref(qualifiedName: $branch)"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}

type ContributionsCollection struct {
	ContributionCalendar struct {
		TotalContributions githubv4.Int
		Weeks              []struct {
			ContributionDays []struct {
				Color             githubv4.String
				ContributionCount githubv4.Int
				Date              githubv4.String
				Weekday           githubv4.Int
			}
			FirstDay githubv4.String
		}
	}
}

type UserData struct {
	User struct {
		Name                    githubv4.String
		Email                   githubv4.String
		ContributionsCollection ContributionsCollection `graphql:"contributionsCollection(from: $from)"`
		IsHireable              githubv4.Boolean
		Repositories            RepositoryConnection `graphql:"repositories(last: 100, isFork: false)"`
		Issues                  Issues               `graphql:"issues(last: 5, filterBy: {since: $sinceOneMonth}, orderBy: {direction:ASC, field: UPDATED_AT})"`
		IssueComments           IssueComments        `graphql:"issueComments(last: 10, orderBy: {direction:ASC, field: UPDATED_AT})"`
	} `graphql:"user(login: $login)"`
}

func (u *user) WriteReadMe(content, commitmsg string) error {
	ctx := context.Background()
	cli := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: u.GHToken},
	)))

	date := time.Now()
	login := string(u.UserInfo.Viewer.Login)
	name := "wakatime-generator-bot"
	email := fmt.Sprintf("41898282+github-actions[bot]@users.noreply.github.com")
	author := &github.CommitAuthor{
		Date:  &date,
		Name:  &name,
		Email: &email,
	}

	c, _, err := cli.Repositories.GetReadme(ctx, login, login, nil)
	if err != nil {
		return err
	}
	oldContent, _ := c.GetContent()
	reg := regexp.MustCompile("<!--START_SECTION:waka-->(.|\n)*<!--END_SECTION:waka-->")
	content = fmt.Sprintf("<!--START_SECTION:waka-->\n%s\n<!--END_SECTION:waka-->", content)
	content = reg.ReplaceAllString(oldContent, content)

	ioutil.WriteFile(os.Getenv("HOME")+"/README_DEBUG.md", []byte(content), 0755)
	// 	return nil
	branch := "main"
	arr := strings.Split(*c.URL, "?")
	if len(arr) > 2 {
		branch = arr[1]
	}
	_, _, err = cli.Repositories.UpdateFile(ctx, string(u.UserInfo.Viewer.Login), string(u.UserInfo.Viewer.Login), "README.md", &github.RepositoryContentFileOptions{
		Message:   &commitmsg,
		SHA:       c.SHA,
		Content:   []byte(content),
		Branch:    &branch,
		Author:    author,
		Committer: author,
	})

	return err
}

func (u *user) sendRequestToGithub(query interface{}, args map[string]interface{}) error {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: u.GHToken},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := githubv4.NewClient(httpClient)

	return client.Query(context.Background(), query, args)
}

const (
	wakatimeBaseURL = "https://wakatime.com/api/v1/"
)

type WakaTimeStats struct {
	Data struct {
		BestDay struct {
			CreatedAt    time.Time `json:"created_at"`
			Date         string    `json:"date"`
			ID           string    `json:"id"`
			ModifiedAt   time.Time `json:"modified_at"`
			Text         string    `json:"text"`
			TotalSeconds float64   `json:"total_seconds"`
		} `json:"best_day"`
		Categories []struct {
			Digital      string  `json:"digital"`
			Hours        int     `json:"hours"`
			Minutes      int     `json:"minutes"`
			Name         string  `json:"name"`
			Percent      float64 `json:"percent"`
			Text         string  `json:"text"`
			TotalSeconds float64 `json:"total_seconds"`
		} `json:"categories"`
		CreatedAt                          time.Time `json:"created_at"`
		DailyAverage                       int       `json:"daily_average"`
		DailyAverageIncludingOtherLanguage int       `json:"daily_average_including_other_language"`
		DaysIncludingHolidays              int       `json:"days_including_holidays"`
		DaysMinusHolidays                  int       `json:"days_minus_holidays"`
		Dependencies                       []struct {
			Digital      string  `json:"digital"`
			Hours        int     `json:"hours"`
			Minutes      int     `json:"minutes"`
			Name         string  `json:"name"`
			Percent      float64 `json:"percent"`
			Text         string  `json:"text"`
			TotalSeconds float64 `json:"total_seconds"`
		} `json:"dependencies"`
		Editors []struct {
			Digital      string  `json:"digital"`
			Hours        int     `json:"hours"`
			Minutes      int     `json:"minutes"`
			Name         string  `json:"name"`
			Percent      float64 `json:"percent"`
			Text         string  `json:"text"`
			TotalSeconds float64 `json:"total_seconds"`
		} `json:"editors"`
		End                                             time.Time `json:"end"`
		Holidays                                        int       `json:"holidays"`
		HumanReadableDailyAverage                       string    `json:"human_readable_daily_average"`
		HumanReadableDailyAverageIncludingOtherLanguage string    `json:"human_readable_daily_average_including_other_language"`
		HumanReadableTotal                              string    `json:"human_readable_total"`
		HumanReadableTotalIncludingOtherLanguage        string    `json:"human_readable_total_including_other_language"`
		ID                                              string    `json:"id"`
		IsAlreadyUpdating                               bool      `json:"is_already_updating"`
		IsCodingActivityVisible                         bool      `json:"is_coding_activity_visible"`
		IsIncludingToday                                bool      `json:"is_including_today"`
		IsOtherUsageVisible                             bool      `json:"is_other_usage_visible"`
		IsStuck                                         bool      `json:"is_stuck"`
		IsUpToDate                                      bool      `json:"is_up_to_date"`
		Languages                                       []struct {
			Digital      string  `json:"digital"`
			Hours        int     `json:"hours"`
			Minutes      int     `json:"minutes"`
			Name         string  `json:"name"`
			Percent      float64 `json:"percent"`
			Text         string  `json:"text"`
			TotalSeconds float64 `json:"total_seconds"`
		} `json:"languages"`
		Machines []struct {
			Digital string `json:"digital"`
			Hours   int    `json:"hours"`
			Machine struct {
				CreatedAt  time.Time `json:"created_at"`
				ID         string    `json:"id"`
				IP         string    `json:"ip"`
				LastSeenAt time.Time `json:"last_seen_at"`
				Name       string    `json:"name"`
				Value      string    `json:"value"`
			} `json:"machine"`
			Minutes      int     `json:"minutes"`
			Name         string  `json:"name"`
			Percent      float64 `json:"percent"`
			Text         string  `json:"text"`
			TotalSeconds float64 `json:"total_seconds"`
		} `json:"machines"`
		ModifiedAt       time.Time `json:"modified_at"`
		OperatingSystems []struct {
			Digital      string  `json:"digital"`
			Hours        int     `json:"hours"`
			Minutes      int     `json:"minutes"`
			Name         string  `json:"name"`
			Percent      float64 `json:"percent"`
			Text         string  `json:"text"`
			TotalSeconds float64 `json:"total_seconds"`
		} `json:"operating_systems"`
		PercentCalculated int         `json:"percent_calculated"`
		Project           interface{} `json:"project"`
		Projects          []struct {
			Digital      string  `json:"digital"`
			Hours        int     `json:"hours"`
			Minutes      int     `json:"minutes"`
			Name         string  `json:"name"`
			Percent      float64 `json:"percent"`
			Text         string  `json:"text"`
			TotalSeconds float64 `json:"total_seconds"`
		} `json:"projects"`
		Range                              string    `json:"range"`
		Start                              time.Time `json:"start"`
		Status                             string    `json:"status"`
		Timeout                            int       `json:"timeout"`
		Timezone                           string    `json:"timezone"`
		TotalSeconds                       float64   `json:"total_seconds"`
		TotalSecondsIncludingOtherLanguage float64   `json:"total_seconds_including_other_language"`
		UserID                             string    `json:"user_id"`
		Username                           string    `json:"username"`
		WritesOnly                         bool      `json:"writes_only"`
	} `json:"data"`
}

func (u *user) getWakaTimeStats() error {
	resp, err := http.Get(fmt.Sprintf("https://wakatime.com/api/v1/users/current/stats/last_7_days?api_key=%s", u.WakaTimeAPIKey))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(bs, u.WakaTimeStats); err != nil {
		return err
	}

	return nil
}
