/*
一些辅助方法
*/

package gopher

import (
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"github.com/jimmykuu/webhelpers"
	"github.com/jimmykuu/wtforms"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	PerPage = 20
)

var (
	store       *sessions.CookieStore
	fileVersion map[string]string = make(map[string]string) // {path: version}
	utils       *Utils
	usersJson   []byte
)

type Utils struct {
}

// 没有http://开头的增加http://
func (u *Utils) Url(url string) string {
	if strings.HasPrefix(url, "http://") {
		return url
	}

	return "http://" + url
}

func (u *Utils) Index(index int) int {
	return index + 1
}

func (u *Utils) FormatDate(t time.Time) string {
	return t.Format(time.RFC822)
}
func (u *Utils) FormatTime(t time.Time) string {
	now := time.Now()
	duration := now.Sub(t)
	if duration.Seconds() < 60 {
		return fmt.Sprintf("刚刚")
	} else if duration.Minutes() < 60 {
		return fmt.Sprintf("%.0f 分钟前", duration.Minutes())
	} else if duration.Hours() < 24 {
		return fmt.Sprintf("%.0f 小时前", duration.Hours())
	}

	t = t.Add(time.Hour * time.Duration(Config.TimeZoneOffset))
	return t.Format("2006-01-02 15:04")
}

func (u *Utils) UserInfo(username string, db *mgo.Database) template.HTML {
	c := db.C(USERS)

	user := User{}
	// 检查用户名
	c.Find(bson.M{"username": username}).One(&user)

	var img string
	if user.Avatar == "" {
		img = string(user.AvatarSVG(48, "class=\"gravatar\""))
	} else {
		img = fmt.Sprintf(`<img class="gravatar img-rounded" src="%s-middle" class="gravatar">`, user.AvatarImgSrc())
	}

	format := `<div>
        <a href="/member/%s">%s</a>
        <h4><a href="/member/%s">%s</a><br><small>%s</small></h4>
	<div class="clearfix">
	</div>
    </div>`

	return template.HTML(fmt.Sprintf(format, username, img, username, username, user.Tagline))
}

func (u *Utils) News(username string, db *mgo.Database) template.HTML {
	c := db.C(USERS)
	user := User{}
	//检查用户名
	c.Find(bson.M{"username": username}).One(&user)
	format := `<div>
		<hr>
		<a href="/member/%s/news#topic">新回复 <span class="badge pull-right">%d</span></a>
		<br>
		<a href="/member/%s/news#at">AT<span class="badge pull-right">%d</span></a>
	</div>
	`
	return template.HTML(fmt.Sprintf(format, username, len(user.RecentReplies), username, len(user.RecentAts)))
}

func (u *Utils) Truncate(html template.HTML, length int) string {
	text := webhelpers.RemoveFormatting(string(html))
	return webhelpers.Truncate(text, length, "...")
}

func (u *Utils) HTML(str string) template.HTML {
	return template.HTML(str)
}

// \n => <br>
func (u *Utils) Br(str string) template.HTML {
	return template.HTML(strings.Replace(str, "\n", "<br>", -1))
}

func (u *Utils) RenderInput(form wtforms.Form, fieldStr string, inputAttrs ...string) template.HTML {
	field, err := form.Field(fieldStr)
	if err != nil {
		panic(err)
	}

	errorClass := ""

	if field.HasErrors() {
		errorClass = " has-error"
	}

	format := `<div class="form-group%s">
        %s
        %s
        %s
    </div>`

	var inputAttrs2 []string = []string{`class="form-control"`}
	inputAttrs2 = append(inputAttrs2, inputAttrs...)

	return template.HTML(
		fmt.Sprintf(format,
			errorClass,
			field.RenderLabel(),
			field.RenderInput(inputAttrs2...),
			field.RenderErrors()))
}

func (u *Utils) RenderInputH(form wtforms.Form, fieldStr string, labelWidth, inputWidth int, inputAttrs ...string) template.HTML {
	field, err := form.Field(fieldStr)
	if err != nil {
		panic(err)
	}

	errorClass := ""

	if field.HasErrors() {
		errorClass = " has-error"
	}
	format := `<div class="form-group%s">
        %s
        <div class="col-lg-%d">
            %s%s
        </div>
    </div>`
	labelClass := fmt.Sprintf(`class="col-lg-%d control-label"`, labelWidth)

	var inputAttrs2 []string = []string{`class="form-control"`}
	inputAttrs2 = append(inputAttrs2, inputAttrs...)

	return template.HTML(
		fmt.Sprintf(format,
			errorClass,
			field.RenderLabel(labelClass),
			inputWidth,
			field.RenderInput(inputAttrs2...),
			field.RenderErrors(),
		))
}

func (u *Utils) AssertUser(i interface{}) *User {
	v, _ := i.(User)
	return &v
}

func (u *Utils) AssertNode(i interface{}) *Node {
	v, _ := i.(Node)
	return &v
}

func (u *Utils) AssertTopic(i interface{}) *Topic {
	v, _ := i.(Topic)
	return &v
}

func (u *Utils) AssertArticle(i interface{}) *Article {
	v, _ := i.(Article)
	return &v
}

func (u *Utils) AssertPackage(i interface{}) *Package {
	v, _ := i.(Package)
	return &v
}

func message(handler *Handler, title string, message string, class string) {
	handler.renderTemplate("message.html", BASE, map[string]interface{}{"title": title, "message": template.HTML(message), "class": class})
}

// 获取链接的页码，默认"?p=1"这种类型
func Page(r *http.Request) (int, error) {
	p := r.FormValue("p")
	page := 1

	if p != "" {
		var err error
		page, err = strconv.Atoi(p)

		if err != nil {
			return 0, err
		}
	}

	return page, nil
}

// 检查一个string元素是否在数组里面
func stringInArray(a []string, x string) bool {
	sort.Strings(a)
	index := sort.SearchStrings(a, x)

	if index == 0 {
		if a[0] == x {
			return true
		}

		return false
	} else if index > len(a)-1 {
		return false
	}

	return true
}

func staticHandler(templateFile string) HandlerFunc {
	return func(handler *Handler) {
		handler.renderTemplate(templateFile, BASE, map[string]interface{}{})
	}
}

func getPage(r *http.Request) (page int, err error) {
	p := r.FormValue("p")
	page = 1

	if p != "" {
		page, err = strconv.Atoi(p)

		if err != nil {
			return
		}
	}

	return
}

//  提取评论中被at的用户名
func findAts(content string) []string {
	allAts := regexp.MustCompile(`@(\S*) `).FindAllStringSubmatch(content, -1)
	var users []string
	for _, v := range allAts {
		users = append(users, v[1])
	}
	return users
}

func searchHandler(handler *Handler) {
	p := handler.Request.FormValue("p")
	page := 1

	if p != "" {
		var err error
		page, err = strconv.Atoi(p)

		if err != nil {
			message(handler, "页码错误", "页码错误", "error")
			return
		}
	}

	q := handler.Request.FormValue("q")

	keywords := strings.Split(q, " ")

	var noSpaceKeywords []string

	for _, keyword := range keywords {
		temp := strings.TrimSpace(keyword)
		if temp != "" {
			noSpaceKeywords = append(noSpaceKeywords, temp)
		}
	}

	var titleConditions []bson.M
	var markdownConditions []bson.M

	for _, keyword := range noSpaceKeywords {
		titleConditions = append(titleConditions, bson.M{"content.title": bson.M{"$regex": bson.RegEx{keyword, "i"}}})
		markdownConditions = append(markdownConditions, bson.M{"content.markdown": bson.M{"$regex": bson.RegEx{keyword, "i"}}})
	}

	c := handler.DB.C(CONTENTS)

	var pagination *Pagination

	if len(noSpaceKeywords) == 0 {
		pagination = NewPagination(c.Find(bson.M{"content.type": TypeTopic}).Sort("-latestrepliedat"), "/search?"+q, PerPage)
	} else {
		pagination = NewPagination(c.Find(bson.M{"$and": []bson.M{
			bson.M{"content.type": TypeTopic},
			bson.M{"$or": []bson.M{
				bson.M{"$and": titleConditions},
				bson.M{"$and": markdownConditions},
			},
			},
		}}).Sort("-latestrepliedat"), "/search?q="+q, PerPage)
	}

	var topics []Topic

	query, err := pagination.Page(page)
	if err != nil {
		message(handler, "页码错误", "页码错误", "error")
		return
	}

	query.(*mgo.Query).All(&topics)

	if err != nil {
		println(err.Error())
	}

	handler.renderTemplate("search.html", BASE, map[string]interface{}{
		"q":          q,
		"topics":     topics,
		"pagination": pagination,
		"page":       page,
		"active":     "topic",
	})
}
