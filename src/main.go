package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/michimani/gotwi"
	"github.com/michimani/gotwi/fields"
	"github.com/michimani/gotwi/resources"
	"github.com/michimani/gotwi/tweets"
	tweetsTypes "github.com/michimani/gotwi/tweets/types"
	"github.com/michimani/gotwi/users"
	usersTypes "github.com/michimani/gotwi/users/types"
)

const (
	Version = "0.1.2"

	DefaultDays    = 30
	DefaultTimeout = 30 * time.Second
	RateLimitSleep = 15 * time.Minute

	envPrefix           = "UNLIKE_"
	EnvOAuthToken       = envPrefix + "OAUTH_TOKEN"
	EnvOAuthTokenSecret = envPrefix + "OAUTH_TOKEN_SECRET"

	ExitSuccess = 0
	ExitUsage   = 2
)

type Opts struct {
	username         string
	days             int
	timeout          time.Duration
	oauthToken       string
	oauthTokenSecret string
}

func getOpts() *Opts {
	isVersion := flag.Bool("V", false, "print version and exit")
	opts := &Opts{
		oauthToken:       os.Getenv(EnvOAuthToken),
		oauthTokenSecret: os.Getenv(EnvOAuthTokenSecret),
	}
	flag.StringVar(&opts.username, "u", "", "username")
	flag.IntVar(&opts.days, "d", DefaultDays, "days")
	flag.DurationVar(&opts.timeout, "t", DefaultTimeout, "timeout")
	flag.Parse()
	if *isVersion {
		fmt.Println(Version)
		os.Exit(ExitSuccess)
	}
	if opts.oauthToken == "" || opts.oauthTokenSecret == "" {
		fmt.Println("oauth token and oauth token secret are required")
		os.Exit(ExitUsage)
	}
	os.Unsetenv(EnvOAuthToken)
	os.Unsetenv(EnvOAuthTokenSecret)
	if opts.days < 0 {
		opts.days = DefaultDays
	}
	return opts
}

type Twitter struct {
	client *gotwi.GotwiClient
}

func NewTwitter(oauthToken, oauthTokenSecret string, timeout time.Duration) (*Twitter, error) {
	client, err := gotwi.NewGotwiClient(&gotwi.NewGotwiClientInput{
		HTTPClient:           &http.Client{Timeout: timeout},
		AuthenticationMethod: gotwi.AuthenMethodOAuth1UserContext,
		OAuthToken:           oauthToken,
		OAuthTokenSecret:     oauthTokenSecret,
	})
	if err != nil {
		return nil, err
	}
	return &Twitter{client: client}, nil
}

func (t *Twitter) LookupUserByUsername(ctx context.Context, username string) (
	resources.User,
	error,
) {
	res, err := users.UserLookupByUsername(
		ctx,
		t.client,
		&usersTypes.UserLookupByUsernameParams{Username: username},
	)
	if err != nil {
		return resources.User{}, err
	}
	return res.Data, nil
}

func (t *Twitter) LookupUserMe(ctx context.Context) (resources.User, error) {
	res, err := users.UserLookupMe(ctx, t.client, &usersTypes.UserLookupMeParams{})
	if err != nil {
		return resources.User{}, err
	}
	return res.Data, nil
}

func (t *Twitter) GetLikedTweets(
	ctx context.Context,
	userID string,
	paginationToken string,
) (
	*tweetsTypes.TweetLikesLikedTweetsResponse,
	error,
) {
	return tweets.TweetLikesLikedTweets(
		ctx,
		t.client,
		&tweetsTypes.TweetLikesLikedTweetsParams{
			ID:              userID,
			PaginationToken: paginationToken,
			TweetFields: fields.TweetFieldList{
				fields.TweetFieldCreatedAt,
				fields.TweetFieldID,
			},
		},
	)
}

func (t *Twitter) Unlike(ctx context.Context, userID, tweetID string) (bool, error) {
	res, err := tweets.TweetLikesDelete(
		ctx,
		t.client,
		&tweetsTypes.TweetLikesDeleteParams{
			ID:      userID,
			TweetID: tweetID,
		},
	)
	if err != nil {
		return false, err
	}
	return !res.Data.Liked, nil
}

func (t *Twitter) CollectLikedTweetsID(
	ctx context.Context,
	userID string,
	days int,
) (
	[]string,
	error,
) {
	ids := []string{}
	token := ""
	for {
		res, err := t.GetLikedTweets(ctx, userID, token)
		if err != nil {
			if trySleepOnError(err) {
				continue
			}
			return nil, err
		}
		point := time.Now().AddDate(0, 0, -days)
		for _, tweet := range res.Data {
			if gotwi.TimeValue(tweet.CreatedAt).After(point) {
				continue
			}
			tweetID := gotwi.StringValue(tweet.ID)
			if tweetID == "" {
				panic("tweet id is empty")
			}
			ids = append(ids, tweetID)
		}
		token = gotwi.StringValue(res.Meta.NextToken)
		if token == "" {
			break
		}
	}
	return ids, nil
}

func (t *Twitter) DeleteLikes(ctx context.Context, userID string, ids []string) error {
	i := 0
	j := 0
	for i < len(ids) {
		id := ids[i]
		res, err := t.Unlike(ctx, userID, id)
		if err != nil {
			if trySleepOnError(err) {
				continue
			}
			log.Printf("failed to unlike tweet %s: %v\n", id, err)
		} else if !res {
			log.Println("unlike failed: ", id)
		} else {
			j++
		}
		i++
		if i%100 == 0 {
			log.Printf("[%d] %d/%d\n", j, i, len(ids))
		}
	}
	return nil
}

func trySleepOnError(err error) bool {
	rv := false
	if err1, ok := err.(net.Error); ok && err1.Timeout() {
		log.Printf("timeout: %v, sleep 30s\n", err1)
		time.Sleep(DefaultTimeout)
		rv = true
	} else if strings.Contains(err.Error(), "httpStatusCode=429") {
		log.Printf("rate limit exceeded: %v, sleep 15m\n", err)
		time.Sleep(RateLimitSleep)
		rv = true
	}
	return rv
}

func fatalOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	opts := getOpts()
	twitter, err := NewTwitter(opts.oauthToken, opts.oauthTokenSecret, opts.timeout)
	fatalOnError(err)
	ctx := context.Background()
	var user resources.User
	if opts.username != "" {
		user, err = twitter.LookupUserByUsername(ctx, opts.username)
	} else {
		user, err = twitter.LookupUserMe(ctx)
	}
	fatalOnError(err)
	userID := gotwi.StringValue(user.ID)
	if userID == "" {
		panic("user id is empty")
	}
	log.Println("user id: ", userID)
	log.Println("collecting liked tweets id...")
	ids, err := twitter.CollectLikedTweetsID(ctx, userID, opts.days)
	fatalOnError(err)
	if len(ids) == 0 {
		log.Println("liked tweets not found")
		os.Exit(ExitSuccess)
	}
	log.Printf("%d likes to delete\n", len(ids))
	fatalOnError(twitter.DeleteLikes(ctx, userID, ids))
	log.Println("done")
}
