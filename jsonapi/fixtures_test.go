package jsonapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gopkg.in/guregu/null.v3/zero"
)

type Magic struct {
	ID MagicID `json:"-"`
}

func (m Magic) GetID() Identifier {
	return Identifier{ID: m.ID.String()}
}

type MagicID string

func (m MagicID) String() string {
	return "This should be visible"
}

type Comment struct {
	ID               int       `json:"-"`
	LID              int       `json:"-"`
	Text             string    `json:"text"`
	SubComments      []Comment `json:"-"`
	SubCommentsEmpty bool      `json:"-"`
}

func (c Comment) GetID() Identifier {
	id := Identifier{ID: fmt.Sprintf("%d", c.ID)}
	if c.LID != 0 {
		id.LID = fmt.Sprintf("%d", c.LID)
	}
	return id
}

func (c *Comment) SetID(ID Identifier) error {
	if ID.ID != "" {
		id, err := strconv.Atoi(ID.ID)
		if err != nil {
			return err
		}
		c.ID = id
	}
	if ID.LID != "" {
		lid, err := strconv.Atoi(ID.LID)
		if err != nil {
			return err
		}
		c.LID = lid
	}

	return nil
}

func (c Comment) GetReferences() []Reference {
	return []Reference{
		{
			Type:        "comments",
			Name:        "comments",
			IsNotLoaded: c.SubCommentsEmpty,
		},
	}
}

func (c Comment) GetReferencedIDs() []ReferenceID {
	result := []ReferenceID{}

	for _, comment := range c.SubComments {
		id := comment.GetID()
		commentID := ReferenceID{Type: "comments", Name: "comments", ID: id.ID, LID: id.LID}
		result = append(result, commentID)
	}

	return result
}

func (c Comment) GetReferencedStructs() []MarshalIdentifier {
	result := []MarshalIdentifier{}

	for _, comment := range c.SubComments {
		result = append(result, comment)
	}

	return result
}

type User struct {
	ID       int    `json:"-"`
	LID      int    `json:"-"`
	Name     string `json:"name"`
	Password string `json:"-"`
}

func (u User) GetID() Identifier {
	id := Identifier{ID: fmt.Sprintf("%d", u.ID)}
	if u.LID != 0 {
		id.LID = fmt.Sprintf("%d", u.LID)
	}
	return id
}

func (u *User) SetID(ID Identifier) error {
	id, err := strconv.Atoi(ID.ID)
	if err != nil {
		return err
	}
	u.ID = id
	lid, err := strconv.Atoi(ID.LID)
	if err != nil {
		return err
	}
	u.LID = lid

	return nil
}

type SimplePost struct {
	ID       string    `json:"-"`
	LID      string    `json:"-"`
	Title    string    `json:"title"`
	Text     string    `json:"text"`
	Internal string    `json:"-"`
	Size     int       `json:"size"`
	Created  time.Time `json:"created-date"`
	Updated  time.Time `json:"updated-date"`
}

func (s SimplePost) GetID() Identifier {
	return Identifier{ID: s.ID, LID: s.LID}
}

func (s *SimplePost) SetID(ID Identifier) error {
	s.ID = ID.ID
	s.LID = ID.LID
	return nil
}

func (s *SimplePost) SetLID(ID string) error {
	s.LID = ID

	return nil
}

type ErrorIDPost struct {
	Error error
}

func (s ErrorIDPost) GetID() Identifier {
	return Identifier{ID: "", LID: ""}
}

func (s *ErrorIDPost) SetID(ID Identifier) error {
	return s.Error
}

type Post struct {
	ID            int           `json:"-"`
	LID           int           `json:"-"`
	Title         string        `json:"title"`
	Comments      []Comment     `json:"-"`
	CommentsIDs   []int         `json:"-"`
	CommentsLIDs  []int         `json:"-"`
	CommentsEmpty bool          `json:"-"`
	Author        *User         `json:"-"`
	AuthorID      sql.NullInt64 `json:"-"`
	AuthorLID     sql.NullInt64 `json:"-"`
	AuthorEmpty   bool          `json:"-"`
}

func (c Post) GetID() Identifier {
	id := Identifier{ID: fmt.Sprintf("%d", c.ID)}
	if c.LID != 0 {
		id.LID = fmt.Sprintf("%d", c.LID)
	}
	return id
}

func (c *Post) SetID(ID Identifier) error {
	if ID.ID != "" {
		id, err := strconv.Atoi(ID.ID)
		if err != nil {
			return err
		}
		c.ID = id
	}
	if ID.LID != "" {
		lid, err := strconv.Atoi(ID.LID)
		if err != nil {
			return err
		}
		c.LID = lid
	}

	return nil
}

func (c *Post) SetLID(stringID string) error {
	if stringID == "" {
		return nil
	}
	var err error
	c.LID, err = strconv.Atoi(stringID)
	if err != nil {
		return err
	}

	return nil
}

func (c Post) GetReferences() []Reference {
	return []Reference{
		{
			Type:        "comments",
			Name:        "comments",
			IsNotLoaded: c.CommentsEmpty,
		},
		{
			Type:        "users",
			Name:        "author",
			IsNotLoaded: c.AuthorEmpty,
		},
	}
}

func (c *Post) SetToOneReferenceID(name string, ID *Identifier) error {
	if name == "author" {
		// Ignore empty author relationships
		if ID != nil {
			if ID.ID != "" {
				intID, err := strconv.ParseInt(ID.ID, 10, 64)
				if err != nil {
					return err
				}
				c.AuthorID = sql.NullInt64{Valid: true, Int64: intID}
			}
			if ID.LID != "" {
				intLID, err := strconv.ParseInt(ID.LID, 10, 64)
				if err != nil {
					return err
				}
				c.AuthorLID = sql.NullInt64{Valid: true, Int64: intLID}
			}
		}

		return nil
	}

	return errors.New("There is no to-one relationship named " + name)
}

func (c *Post) SetToManyReferenceIDs(name string, IDs []Identifier) error {
	if name == "comments" {
		var commentsIDs []int
		var commentsLIDs []int

		for _, ID := range IDs {
			if ID.ID != "" {
				intID, err := strconv.ParseInt(ID.ID, 10, 64)
				if err != nil {
					return err
				}

				commentsIDs = append(commentsIDs, int(intID))
			}
			if ID.LID != "" {
				intLID, err := strconv.ParseInt(ID.LID, 10, 64)
				if err != nil {
					return err
				}

				commentsLIDs = append(commentsLIDs, int(intLID))
			}
		}

		c.CommentsIDs = commentsIDs
		c.CommentsLIDs = commentsLIDs

		return nil
	}

	return errors.New("There is no to-many relationship named " + name)
}

func (c *Post) SetReferencedIDs(ids []ReferenceID) error {
	for _, reference := range ids {
		intID, err := strconv.ParseInt(reference.ID, 10, 64)
		if err != nil {
			return err
		}

		switch reference.Name {
		case "comments":
			c.CommentsIDs = append(c.CommentsIDs, int(intID))
		case "author":
			c.AuthorID = sql.NullInt64{Valid: true, Int64: intID}
		}
	}

	return nil
}

func (c Post) GetReferencedIDs() []ReferenceID {
	result := []ReferenceID{}

	if c.Author != nil {
		id := c.Author.GetID()
		authorID := ReferenceID{Type: "users", Name: "author", ID: id.ID, LID: id.LID}
		result = append(result, authorID)
	} else if c.AuthorID.Valid {
		authorID := ReferenceID{Type: "users", Name: "author", ID: fmt.Sprintf("%d", c.AuthorID.Int64)}
		result = append(result, authorID)
	}

	if len(c.Comments) > 0 {
		for _, comment := range c.Comments {
			id := comment.GetID()
			result = append(result, ReferenceID{Type: "comments", Name: "comments", ID: id.ID, LID: id.LID})
		}
	} else if len(c.CommentsIDs) > 0 {
		for _, commentID := range c.CommentsIDs {
			result = append(result, ReferenceID{Type: "comments", Name: "comments", ID: fmt.Sprintf("%d", commentID)})
		}
	}

	return result
}

func (c Post) GetReferencedStructs() []MarshalIdentifier {
	result := []MarshalIdentifier{}

	if c.Author != nil {
		result = append(result, c.Author)
	}

	for key := range c.Comments {
		result = append(result, c.Comments[key])
	}

	return result
}

func (c *Post) SetReferencedStructs(references []UnmarshalIdentifier) error {
	return nil
}

type AnotherPost struct {
	ID       int   `json:"-"`
	LID      int   `json:"-"`
	AuthorID int   `json:"-"`
	Author   *User `json:"-"`
}

func (p AnotherPost) GetID() Identifier {
	id := Identifier{ID: fmt.Sprintf("%d", p.ID)}
	if p.LID != 0 {
		id.LID = fmt.Sprintf("%d", p.LID)
	}
	return id
}

func (p AnotherPost) GetReferences() []Reference {
	return []Reference{
		{
			Type: "users",
			Name: "author",
		},
	}
}

func (p AnotherPost) GetReferencedIDs() []ReferenceID {
	result := []ReferenceID{}

	if p.AuthorID != 0 {
		result = append(result, ReferenceID{ID: fmt.Sprintf("%d", p.AuthorID), Name: "author", Type: "users"})
	}

	return result
}

type ZeroPost struct {
	ID    string     `json:"-"`
	LID   string     `json:"-"`
	Title string     `json:"title"`
	Value zero.Float `json:"value"`
}

func (z ZeroPost) GetID() Identifier {
	return Identifier{ID: z.ID, LID: z.LID}
}

type ZeroPostPointer struct {
	ID    string      `json:"-"`
	LID   string      `json:"-"`
	Title string      `json:"title"`
	Value *zero.Float `json:"value"`
}

func (z ZeroPostPointer) GetID() Identifier {
	return Identifier{ID: z.ID, LID: z.LID}
}

type Question struct {
	ID                  string         `json:"-"`
	Text                string         `json:"text"`
	InspiringQuestionID sql.NullString `json:"-"`
	InspiringQuestion   *Question      `json:"-"`
}

func (q Question) GetID() Identifier {
	return Identifier{ID: q.ID, LID: ""}
}

func (q Question) GetReferences() []Reference {
	return []Reference{
		{
			Type: "questions",
			Name: "inspiringQuestion",
		},
	}
}

func (q Question) GetReferencedIDs() []ReferenceID {
	result := []ReferenceID{}

	if q.InspiringQuestionID.Valid {
		result = append(result, ReferenceID{ID: q.InspiringQuestionID.String, Name: "inspiringQuestion", Type: "questions"})
	}

	return result
}

func (q Question) GetReferencedStructs() []MarshalIdentifier {
	result := []MarshalIdentifier{}

	if q.InspiringQuestion != nil {
		result = append(result, *q.InspiringQuestion)
	}

	return result
}

type Identity struct {
	ID     int64    `json:"-"`
	LID    int64    `json:"-"`
	Scopes []string `json:"scopes"`
}

func (i Identity) GetID() Identifier {
	id := Identifier{ID: fmt.Sprintf("%d", i.ID)}
	if i.LID != 0 {
		id.LID = fmt.Sprintf("%d", i.LID)
	}
	return id
}

func (i *Identity) SetID(ID Identifier) error {
	if ID.ID != "" {
		id, err := strconv.Atoi(ID.ID)
		if err != nil {
			return err
		}
		i.ID = int64(id)
	}
	if ID.LID != "" {
		lid, err := strconv.Atoi(ID.LID)
		if err != nil {
			return err
		}
		i.LID = int64(lid)
	}

	return nil
}

type Unicorn struct {
	UnicornID int64    `json:"unicorn_id"` // annotations are ignored
	Scopes    []string `json:"scopes"`
}

func (u Unicorn) GetID() Identifier {
	return Identifier{ID: "magicalUnicorn", LID: ""}
}

type NumberPost struct {
	ID             string `json:"-"`
	LID            string `json:"-"`
	Title          string
	Number         int64
	UnsignedNumber uint64
}

func (n NumberPost) GetID() Identifier {
	return Identifier{ID: n.ID, LID: n.LID}
}

func (n *NumberPost) SetID(ID Identifier) error {
	n.ID = ID.ID
	n.LID = ID.LID
	return nil
}

type SQLNullPost struct {
	ID     string      `json:"-"`
	LID    string      `json:"-"`
	Title  zero.String `json:"title"`
	Likes  zero.Int    `json:"likes"`
	Rating zero.Float  `json:"rating"`
	IsCool zero.Bool   `json:"isCool"`
	Today  zero.Time   `json:"today"`
}

func (s SQLNullPost) GetID() Identifier {
	return Identifier{ID: s.ID, LID: s.LID, Name: "sqlNullPosts"}
}

func (s *SQLNullPost) SetID(ID Identifier) error {
	s.ID = ID.ID
	s.LID = ID.LID
	return nil
}

type RenamedPostWithEmbedding struct {
	Embedded SQLNullPost
	ID       string `json:"-"`
	LID      string `json:"-"`
	Another  string `json:"another"`
	Field    string `json:"foo"`
	Other    string `json:"bar-bar"`
	Ignored  string `json:"-"`
}

func (p RenamedPostWithEmbedding) GetID() Identifier {
	return Identifier{ID: p.ID, LID: p.LID}
}

func (p *RenamedPostWithEmbedding) SetID(ID Identifier) error {
	p.ID = ID.ID
	p.LID = ID.LID
	return nil
}

type RenamedComment struct {
	Data string
}

func (r RenamedComment) GetID() Identifier {
	return Identifier{ID: "666", LID: "", Name: "renamed-comments"}
}

type CompleteServerInformation struct{}

const baseURL = "http://my.domain"
const prefix = "v1"

func (i CompleteServerInformation) GetBaseURL() string {
	return baseURL
}

func (i CompleteServerInformation) GetPrefix() string {
	return prefix
}

type BaseURLServerInformation struct{}

func (i BaseURLServerInformation) GetBaseURL() string {
	return baseURL
}

func (i BaseURLServerInformation) GetPrefix() string {
	return ""
}

type PrefixServerInformation struct{}

func (i PrefixServerInformation) GetBaseURL() string {
	return ""
}

func (i PrefixServerInformation) GetPrefix() string {
	return prefix
}

type CustomLinksPost struct{}

func (n CustomLinksPost) GetID() Identifier {
	return Identifier{ID: "someID", LID: "", Name: "posts"}
}

func (n *CustomLinksPost) SetID(ID Identifier) error {
	return nil
}

func (n CustomLinksPost) GetCustomLinks(base string) Links {
	return Links{
		"nothingInHere": Link{},
		"someLink":      Link{Href: base + `/someLink`},
		"otherLink": Link{
			Href: base + `/otherLink`,
			Meta: Meta{
				"method": "GET",
			},
		},
	}
}

type CustomResourceMetaPost struct{}

func (n CustomResourceMetaPost) GetID() Identifier {
	return Identifier{ID: "someID", LID: "", Name: "posts"}
}

func (n *CustomResourceMetaPost) SetID(ID Identifier) error {
	return nil
}

func (n CustomResourceMetaPost) Meta() Meta {
	return Meta{"access_count": 15}
}

type CustomMetaPost struct{}

func (n CustomMetaPost) GetID() Identifier {
	return Identifier{ID: "someID", LID: "", Name: "posts"}
}

func (n *CustomMetaPost) SetID(ID Identifier) error {
	return nil
}

func (n CustomMetaPost) GetReferences() []Reference {
	return []Reference{
		{
			Type:        "users",
			Name:        "author",
			IsNotLoaded: true,
		},
	}
}

func (n CustomMetaPost) GetReferencedIDs() []ReferenceID {
	return nil
}

func (n CustomMetaPost) GetCustomMeta(linkURL string) map[string]Meta {
	meta := map[string]Meta{
		"author": {
			"someMetaKey":      "someMetaValue",
			"someOtherMetaKey": "someOtherMetaValue",
		},
	}
	return meta
}

type NoRelationshipPosts struct{}

func (n NoRelationshipPosts) GetID() Identifier {
	return Identifier{ID: "someID", LID: "", Name: "posts"}
}

func (n *NoRelationshipPosts) SetID(ID Identifier) error {
	return nil
}

type ErrorRelationshipPosts struct{}

func (e ErrorRelationshipPosts) GetID() Identifier {
	return Identifier{ID: "errorID", LID: "", Name: "posts"}
}

func (e *ErrorRelationshipPosts) SetID(ID Identifier) error {
	return nil
}

func (e ErrorRelationshipPosts) SetToOneReferenceID(name string, ID *Identifier) error {
	return errors.New("this never works")
}

func (e ErrorRelationshipPosts) SetToManyReferenceIDs(name string, IDs []Identifier) error {
	return errors.New("this also never works")
}

type Image struct {
	ID    string      `json:"-"`
	LID   string      `json:"-"`
	Ports []ImagePort `json:"image-ports"`
}

func (i Image) GetID() Identifier {
	return Identifier{ID: i.ID, LID: i.LID}
}

func (i *Image) SetID(ID Identifier) error {
	i.ID = ID.ID
	i.LID = ID.LID
	return nil
}

type ImagePort struct {
	Protocol string `json:"protocol"`
	Number   int    `json:"number"`
}

type Article struct {
	IDs          []string         `json:"-"`
	Type         string           `json:"-"`
	Name         string           `json:"-"`
	Relationship RelationshipType `json:"-"`
}

func (a Article) GetID() Identifier {
	return Identifier{ID: "id", LID: ""}
}

func (a Article) GetReferences() []Reference {
	return []Reference{{Type: a.Type, Name: a.Name, Relationship: a.Relationship}}
}

func (a Article) GetReferencedIDs() []ReferenceID {
	referenceIDs := []ReferenceID{}

	for _, id := range a.IDs {
		referenceIDs = append(referenceIDs, ReferenceID{ID: id, Type: a.Type, Name: a.Name, Relationship: a.Relationship})
	}

	return referenceIDs
}

type DeepDedendencies struct {
	ID            string             `json:"-"`
	Relationships []DeepDedendencies `json:"-"`
}

func (d DeepDedendencies) GetID() Identifier {
	return Identifier{ID: d.ID, LID: "", Name: "deep"}
}

func (d DeepDedendencies) GetReferences() []Reference {
	return []Reference{{Type: "deep", Name: "deps"}}
}

func (d DeepDedendencies) GetReferencedIDs() []ReferenceID {
	references := make([]ReferenceID, 0, len(d.Relationships))

	for _, r := range d.Relationships {
		references = append(references, ReferenceID{ID: r.ID, Type: "deep", Name: "deps"})
	}

	return references
}

func (d DeepDedendencies) GetReferencedStructs() []MarshalIdentifier {
	var structs []MarshalIdentifier

	for _, r := range d.Relationships {
		structs = append(structs, r)
		structs = append(structs, r.GetReferencedStructs()...)
	}

	return structs
}

type SimplePostWithMetadata struct {
	SimplePost
	ResourceMetadata map[string]interface{}
}

func (p *SimplePostWithMetadata) SetResourceMeta(r json.RawMessage) error {
	return json.Unmarshal(r, &p.ResourceMetadata)
}
