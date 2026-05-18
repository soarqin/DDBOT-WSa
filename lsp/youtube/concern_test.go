package youtube

import (
	"context"
	"github.com/cnxysoft/DDBOT-WSa/internal/test"
	"github.com/cnxysoft/DDBOT-WSa/lsp/concern"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestConcern(t *testing.T) {
	test.InitBuntdb(t)
	defer test.CloseBuntdb(t)

	testEventChan := make(chan concern.Event, 16)
	testNotifyChan := make(chan concern.Notify)

	c := NewConcern(testNotifyChan)

	assert.NotNil(t, c.GetStateManager())

	c.StateManager.UseNotifyGeneratorFunc(c.notifyGenerator())
	c.StateManager.UseFreshFunc(func(ctx context.Context, eventChan chan<- concern.Event) {
		for {
			select {
			case e := <-testEventChan:
				if e != nil {
					eventChan <- e
				}
			case <-ctx.Done():
				return
			}
		}
	})

	assert.Nil(t, c.StateManager.Start())
	defer c.Stop()
	defer close(testEventChan)

	_, err := c.ParseId(test.NAME1)
	assert.Nil(t, err)

	err = c.StateManager.AddInfo(&Info{
		UserInfo: UserInfo{
			ChannelId:   test.NAME1,
			ChannelName: test.NAME1,
		},
	})
	assert.Nil(t, err)

	_, err = c.StateManager.AddGroupConcern(test.G1, test.NAME1, Live)
	assert.Nil(t, err)

	_, err = c.StateManager.AddGroupConcern(test.G2, test.NAME1, Live)
	assert.Nil(t, err)

	identityInfo, err := c.Get(test.NAME1)
	assert.Nil(t, err)
	assert.EqualValues(t, test.NAME1, identityInfo.GetUid())

	assert.NotNil(t, c.GetGroupConcernConfig(test.G1, test.NAME1))

	testEventChan <- &VideoInfo{
		UserInfo: UserInfo{
			ChannelId:   test.NAME1,
			ChannelName: test.NAME1,
		},
		VideoType:         VideoType_Live,
		VideoStatus:       VideoStatus_Living,
		liveStatusChanged: true,
	}

	time.Sleep(time.Millisecond * 500)

	var g1 = false
	var g2 = false

	for i := 0; i < 2; i++ {
		select {
		case notify := <-testNotifyChan:
			if notify.GetGroupCode() == test.G1 {
				g1 = true
				assert.Equal(t, test.G1, notify.GetGroupCode())
				assert.Equal(t, test.NAME1, notify.GetUid())
			}
			if notify.GetGroupCode() == test.G2 {
				g2 = true
				assert.Equal(t, test.G2, notify.GetGroupCode())
				assert.Equal(t, test.NAME1, notify.GetUid())
			}
		case <-time.After(time.Second):
			assert.Fail(t, "no notify received")
		}
	}

	assert.True(t, g1)
	assert.True(t, g2)

	select {
	case <-testNotifyChan:
		assert.Fail(t, "should no notify received")
	case <-time.After(time.Second):

	}

	_, err = c.Remove(nil, test.G1, test.NAME1, Live)
	assert.Nil(t, err)
	_, err = c.Remove(nil, test.G2, test.NAME1, Live)
	assert.Nil(t, err)
}

func TestFreshInfoWithoutOldInfoDoesNotPanic(t *testing.T) {
	test.InitBuntdb(t)
	defer test.CloseBuntdb(t)

	oldTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `<!doctype html><script>var ytInitialData = {"metadata":{"channelMetadataRenderer":{"title":"Test Channel"}},"contents":{"gridVideoRenderer":{"videoId":"test_live_id","title":{"simpleText":"Test Live"},"thumbnail":{"thumbnails":[{"url":"https://example.com/cover.jpg","height":180}]},"thumbnailOverlays":[{"thumbnailOverlayTimeStatusRenderer":{"text":{"accessibility":{"accessibilityData":{"label":"LIVE"}}},"style":"LIVE"}}]}}};</script>`
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})
	defer func() {
		http.DefaultTransport = oldTransport
	}()

	c := NewConcern(make(chan concern.Notify))

	var infos []*VideoInfo
	assert.NotPanics(t, func() {
		var err error
		infos, err = c.freshInfo(test.NAME1)
		assert.Nil(t, err)
	})
	if !assert.Len(t, infos, 1) {
		return
	}
	assert.True(t, infos[0].IsLiving())
	assert.True(t, infos[0].LiveStatusChanged())
}
