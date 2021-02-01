package main

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/shurcooL/githubv4"
)

var (
	ghToken        = ""
	wakaTimeAPIKey = ""
	timezone       = ""
	timeLocation   = time.Local
)

func init() {
	var err error

	ghToken = os.Getenv("GH_TOKEN")
	wakaTimeAPIKey = os.Getenv("WAKATIME_API_KEY")
	timezone = os.Getenv("TIMEZONE")
	if timezone != "" {
		timeLocation, err = time.LoadLocation(timezone)
		if err != nil {
			panic(err)
		}
	}
}

func main() {
	u := Init(ghToken, wakaTimeAPIKey)
	if timezone == "" {
		timezone = time.Local.String()
	}

	langMap := map[string]int{}
	mostlyLangName := ""
	mostlyLangNum := 0
	private, public := 0, 0
	u.RepositoryCommitInfo = make([]*RepositoryCommitInfo, len(u.UserData.User.Repositories.Edges))
	for i, edge := range u.UserData.User.Repositories.Edges {
		node := edge.Node
		rci := &RepositoryCommitInfo{}
		if err := u.sendRequestToGithub(rci, map[string]interface{}{
			"owner":  githubv4.String(node.Owner.Login),
			"name":   githubv4.String(node.Name),
			"last":   githubv4.Int(100),
			"id":     u.UserInfo.Viewer.ID.(string),
			"branch": githubv4.String(node.DefaultBranchRef.Name),
			"since":  githubv4.GitTimestamp{Time: GetFirstDateOfWeek()},
		}); err != nil {
			panic(err)
		}
		u.RepositoryCommitInfo[i] = rci

		langName := string(node.PrimaryLanguage.Name)
		if langName == "" {
			langName = "Other"
		}

		langMap[langName]++
		if langMap[langName] > mostlyLangNum {
			mostlyLangName = langName
			mostlyLangNum = langMap[langName]
		}
		if node.IsPrivate {
			private++
		} else {
			public++
		}
	}

	content := ""
	content += GenIssues(u)

	// Section: My Github Data
	content += GenGithubData(u, public, private)

	// Section I'm an Early And Mostly Productive
	content += GenCommitsInfo(u)

	// Section: I Mostly code in `LANG`
	content += GenMostly(u, langMap, mostlyLangName)

	if u.WakaTimeAPIKey != "" {
		// Section: This Week I Spent My Time On
		content += GenWakaTimeStats(u)
	}

	err := u.WriteReadMe(content, "update README.md.")

	if err != nil {
		panic(err)
	}

	fmt.Println(content)
	fmt.Println("build readme successful!")
}

func GenIssues(u *user) string {
	content := ""
	content += "\n"

	for _, edge := range u.UserData.User.Issues.Edges {
		node := edge.Node
		fmt.Println(node.Title, node.UpdatedAt)
		content = fmt.Sprintf(
			"%s\n\n",
			fmt.Sprintf("[%s](%s)", string(node.Title), node.URL.String()),
		) + content
	}

	return content
}

func GetFirstDateOfWeek() time.Time {
	now := time.Now()

	offset := int(time.Monday - now.Weekday())
	if offset > 0 {
		offset = -6
	}

	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, timeLocation).AddDate(0, 0, offset)
}

func GenGithubData(u *user, public, private int) string {
	content := ""
	totalDiskUsage := float64(u.UserData.User.Repositories.TotalDiskUsage) / 1024
	totalContributions := u.UserData.User.ContributionsCollection.ContributionCalendar.TotalContributions
	content += "**🐱 My Github Data**\n"
	content += fmt.Sprintf("> 🏆 %d Contributions in the Year %d\n >\n", totalContributions, time.Now().Year())
	content += fmt.Sprintf("> 📦 package %.2f MB Used in Github's Storage\n >\n", totalDiskUsage)
	if u.UserData.User.IsHireable {
		content += fmt.Sprintf("> 💼 Opted to Hire\n >\n")
	} else {
		content += fmt.Sprintf("> 🚫 Not Opted to Hire\n >\n")
	}
	content += fmt.Sprintf("> 🚪 %d Public Repositories\n >\n", public)
	content += fmt.Sprintf("> 🔑 %d Pirvate Repositories\n >\n", private)

	return content
}

func GenMostly(u *user, langMap map[string]int, mostlyLangName string) string {
	content := ""
	totalRepo := float64(u.UserData.User.Repositories.TotalCount)
	content += fmt.Sprintf("\n")
	content += fmt.Sprintf("**❤ I Mostly Code in %s**\n", mostlyLangName)
	content += fmt.Sprintf("\n")
	content += fmt.Sprintf("```text\n")
	langList := []string{}

	for k := range langMap {
		langList = append(langList, k)
	}

	sort.Strings(langList)

	for _, langName := range langList {
		repoNum := langMap[langName]
		per := float64(repoNum) / totalRepo

		content += fmt.Sprintf(
			"%s%s%s\t%.2f%%\n",
			TextWithTab(langName, 2),
			TextWithTab(fmt.Sprintf("%d repos", repoNum), 2),
			GenProgressBar(per),
			per*100,
		)
	}
	content += fmt.Sprintf("```\n")
	return content
}

func GenWakaTimeStats(u *user) string {
	content := ""
	content += "\n"
	content += fmt.Sprintf("**📊 This Week I Spent My Time On**\n")
	content += fmt.Sprintf("```text\n")
	for _, project := range u.WakaTimeStats.Data.Projects {
		if project.Name == "Unknown Project" {
			continue
		}
		content += fmt.Sprintf(
			"%s%s%s\t%.2f%%\n",
			TextWithTab(project.Name, 2),
			TextWithTab(project.Text, 2),
			GenProgressBar(project.Percent/100),
			project.Percent,
		)
	}
	content += fmt.Sprintf("```\n")
	return content
}

func GenCommitsInfo(u *user) string {
	weekdayMap := map[string]int{
		"Monday":    0,
		"Tuesday":   0,
		"Wednesday": 0,
		"Thursday":  0,
		"Friday":    0,
		"Saturday":  0,
		"Sunday":    0,
	}
	weekdayList := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}

	timeMap := map[string]int{
		"Morning": 0,
		"Daytime": 0,
		"Evening": 0,
		"Night":   0,
	}
	timeList := []string{"Morning", "Daytime", "Evening", "Night"}

	repoMap := map[string]int{}
	repoList := []string{}

	total := 0
	maxDay := ""
	maxRepo := ""
	for _, rci := range u.RepositoryCommitInfo {
		repoName := string(rci.Repository.Name)
		repoList = append(repoList, repoName)
		for _, d := range rci.Repository.Ref.Target.Commit.History.Edges {
			date := d.Node.CommittedDate.In(timeLocation)
			w := date.Weekday().String()
			t := date.Hour()
			switch {
			case t < 6:
				timeMap["Night"]++
			case t < 12:
				timeMap["Morning"]++
			case t < 18:
				timeMap["Daytime"]++
			case t < 24:
				timeMap["Evening"]++
			}

			weekdayMap[w]++
			if weekdayMap[w] > weekdayMap[maxDay] {
				maxDay = w
			}

			repoMap[repoName]++
			if repoMap[repoName] > repoMap[maxRepo] {
				maxRepo = repoName
			}

			total++
		}
	}

	content := ""

	content += "\n"
	content += "**I'm an Early 🐤** \n"
	content += "```text\n"
	for _, i := range timeList {
		per := float64(timeMap[i]) / float64(total)

		content += fmt.Sprintf(
			"%s%s%s\t%.2f%%\n",
			TextWithTab(i, 2),
			TextWithTab(fmt.Sprintf("%d commits", timeMap[i]), 2),
			GenProgressBar(per),
			per*100,
		)
	}
	content += "```\n"

	content += "\n"
	content += fmt.Sprintf("**📅 I'm Most Productive on %s**\n", maxDay)
	content += "```text\n"
	for _, i := range weekdayList {
		per := float64(weekdayMap[i]) / float64(total)

		content += fmt.Sprintf(
			"%s%s%s\t%.2f%%\n",
			TextWithTab(i, 2),
			TextWithTab(fmt.Sprintf("%d commits", weekdayMap[i]), 2),
			GenProgressBar(per),
			per*100,
		)
	}
	content += "```\n"

	content += "\n"
	content += fmt.Sprintf("**📽 I'm Most Contribute to %s**\n", maxRepo)
	content += "```text\n"
	sort.Strings(repoList)
	for _, repo := range repoList {
		if repoMap[repo] == 0 {
			continue
		}
		per := float64(repoMap[repo]) / float64(total)
		content += fmt.Sprintf(
			"%s%s%s\t%.2f%%\n",
			TextWithTab(repo, 2),
			TextWithTab(fmt.Sprintf("%d commits", repoMap[repo]), 2),
			GenProgressBar(per),
			per*100,
		)
	}
	content += "```\n"
	content += "\n"

	return content
}

func TextWithTab(text string, maxTab int) string {
	if len(text) > 15 {
		bs := []byte(text)
		bs = bs[:15]
		bs[12], bs[13], bs[14] = '.', '.', '.'
		text = string(bs)
	}

	tab := ""
	maxTab = maxTab - len(text)/8
	tab += text
	for i := 0; i < maxTab; i++ {
		tab += "\t"
	}

	return tab
}

func GenProgressBar(per float64) string {
	block := ""
	n := int(per * 25)
	for i := 0; i < 25; i++ {
		if i < n {
			block += "#"
		} else {
			block += "-"
		}
	}
	return block
}
