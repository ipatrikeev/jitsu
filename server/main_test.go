package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/jitsucom/jitsu/server/testsuite"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/jitsucom/jitsu/server/logging"
	"github.com/jitsucom/jitsu/server/middleware"
	"github.com/jitsucom/jitsu/server/telemetry"
	"github.com/jitsucom/jitsu/server/test"
	"github.com/jitsucom/jitsu/server/uuid"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"gopkg.in/segmentio/analytics-go.v3"
)

func SetTestDefaultParams() {
	viper.Set("log.path", "")
	viper.Set("api_keys", `{"tokens":[{"id":"id1","client_secret":"c2stoken","server_secret":"s2stoken","origins":["whiteorigin*"]}]}`)
	viper.Set("server.log.path", "")
}

func TestCors(t *testing.T) {
	uuid.InitMock()
	binding.EnableDecoderUseNumber = true

	SetTestDefaultParams()
	tests := []struct {
		Name       string
		ReqUrn     string
		ReqOrigin  string
		XAuthToken string

		ExpectedCorsHeaderValue string
		ResponseCode            int
	}{
		{
			"Wrong token in event url",
			"/api/v1/event?token=wrongtoken",
			"",
			"",
			"",
			401,
		},
		{
			"Wrong token in random url",
			"/api.dadaba?p_dn1231dada=wrongtoken",
			"",
			"",
			"",
			401,
		},
		{
			"Wrong token in header event url",
			"/api/v1/event",
			"",
			"wrongtoken",
			"",
			401,
		},
		{
			"Wrong token in header random url",
			"/api.d2d3ba",
			"",
			"wrongtoken",
			"",
			401,
		},
		{
			"Wrong origin with token in event url",
			"/api/v1/event?token=c2stoken",
			"origin.com",
			"",
			"",
			200,
		},
		{
			"Wrong origin with token in random url",
			"/api.djla9a?p_dlkiud7=wrongtoken",
			"origin.com",
			"",
			"",
			401,
		},
		{
			"Wrong origin with token in header event url",
			"/api/v1/event",
			"origin.com",
			"c2stoken",
			"",
			200,
		},
		{
			"Wrong origin with token in header random url",
			"/api.dn12o31",
			"origin.com",
			"c2stoken",
			"",
			200,
		},
		{
			"Ok origin with token in event url",
			"/api/v1/event?token=c2stoken",
			"https://whiteorigin.com",
			"",
			"https://whiteorigin.com",
			200,
		},
		{
			"Ok origin with token in random url",
			"/api.dn1239?p_km12418hdasd=c2stoken",
			"https://whiteorigin.com",
			"",
			"https://whiteorigin.com",
			200,
		},
		{
			"Ok origin with token in header event url",
			"/api/v1/event",
			"http://whiteoriginmy.com",
			"c2stoken",
			"http://whiteoriginmy.com",
			200,
		},
		{
			"Ok origin with token in header random url",
			"/api.i12310h",
			"http://whiteoriginmy.com",
			"c2stoken",
			"http://whiteoriginmy.com",
			200,
		},
		{
			"S2S endpoint without cors",
			"/api/v1/s2s/event?token=wrongtoken",
			"",
			"",
			"",
			200,
		},
		{
			"static endpoint /t",
			"/t/path",
			"",
			"",
			"*",
			200,
		},
		{
			"static endpoint /s",
			"/s/path",
			"",
			"",
			"*",
			200,
		},
		{
			"static endpoint /p",
			"/p/path",
			"",
			"",
			"*",
			200,
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			testSuite := testsuite.NewSuiteBuilder(t).Build(t)
			defer testSuite.Close()

			//check http OPTIONS
			optReq, err := http.NewRequest(http.MethodOptions, "http://"+testSuite.HTTPAuthority()+tt.ReqUrn, nil)
			require.NoError(t, err)
			if tt.ReqOrigin != "" {
				optReq.Header.Add("Origin", tt.ReqOrigin)
			}
			if tt.XAuthToken != "" {
				optReq.Header.Add(middleware.TokenHeaderName, tt.XAuthToken)
			}
			optResp, err := http.DefaultClient.Do(optReq)
			require.NoError(t, err)

			require.Equal(t, tt.ResponseCode, optResp.StatusCode)

			require.Equal(t, tt.ExpectedCorsHeaderValue, optResp.Header.Get("Access-Control-Allow-Origin"), "Cors header ACAO values aren't equal")
			optResp.Body.Close()
		})
	}
}

func TestEventEndpoint(t *testing.T) {
	uuid.InitMock()
	binding.EnableDecoderUseNumber = true

	SetTestDefaultParams()
	tests := []test.Integration{
		{
			"Unauthorized c2s endpoint",
			"/api/v1/event?token=wrongtoken",
			"test_data/event_input_1.0.json",
			"",
			"",
			http.StatusUnauthorized,
			`{"message":"The token is not found","error":""}`,
		},
		{
			"Unauthorized s2s endpoint",
			"/api/v1/s2s/event?token=c2stoken",
			"test_data/api_event_input_1.0.json",
			"",
			"",
			http.StatusUnauthorized,
			`{"message":"The token isn't a server token. Please use s2s integration token","error":""}`,
		},
		{
			"Unauthorized c2s endpoint with s2s token",
			"/api/v1/event?token=s2stoken",
			"test_data/event_input_1.0.json",
			"",
			"",
			http.StatusUnauthorized,
			`{"message":"The token is not found","error":""}`,
		},
		{
			"C2S event 1.0 consuming test",
			"/api/v1/event?token=c2stoken",
			"test_data/event_input_1.0.json",
			"test_data/fact_output_1.0.json",
			"",
			http.StatusOK,
			"",
		},
		{
			"S2S API 1.0 event consuming test",
			"/api/v1/s2s/event",
			"test_data/api_event_input_1.0.json",
			"test_data/api_fact_output_1.0.json",
			"s2stoken",
			http.StatusOK,
			"",
		},
		{
			"S2S API malformed event test",
			"/api/v1/s2s/event",
			"test_data/malformed_input.json",
			"",
			"s2stoken",
			http.StatusBadRequest,
			`{"message":"Error parsing events body: error parsing HTTP body: invalid character 'a' looking for beginning of object key string","error":""}`,
		},
		{
			"Randomized c2s endpoint 1.0",
			"/api.dhb31?p_neoq231=c2stoken",
			"test_data/event_input_1.0.json",
			"test_data/fact_output_1.0.json",
			"",
			http.StatusOK,
			"",
		},
		{
			"C2S event 2.0 consuming test",
			"/api/v1/event?token=c2stoken",
			"test_data/event_input_2.0.json",
			"test_data/fact_output_2.0.json",
			"",
			http.StatusOK,
			"",
		},
		{
			"S2S API 2.0 event consuming test",
			"/api/v1/s2s/event",
			"test_data/api_event_input_2.0.json",
			"test_data/api_fact_output_2.0.json",
			"s2stoken",
			http.StatusOK,
			"",
		},
		{
			"Mobile API single event",
			"/api/v1/event",
			"test_data/mobile_event_input.json",
			"test_data/mobile_fact_output.json",
			"c2stoken",
			http.StatusOK,
			"",
		},
		{
			"Mobile API events array",
			"/api/v1/event",
			"test_data/mobile_events_array_input.json",
			"test_data/mobile_facts_array_output.json",
			"c2stoken",
			http.StatusOK,
			"",
		},
		{
			"Mobile API events with template",
			"/api/v1/event",
			"test_data/mobile_events_template_input.json",
			"test_data/mobile_facts_template_output.json",
			"c2stoken",
			http.StatusOK,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			testSuite := testsuite.NewSuiteBuilder(t).Build(t)
			defer testSuite.Close()

			b, err := ioutil.ReadFile(tt.ReqBodyPath)
			require.NoError(t, err)

			//check http POST
			apiReq, err := http.NewRequest("POST", "http://"+testSuite.HTTPAuthority()+tt.ReqUrn, bytes.NewBuffer(b))
			require.NoError(t, err)
			if tt.XAuthToken != "" {
				apiReq.Header.Add(middleware.TokenHeaderName, tt.XAuthToken)
			}
			apiReq.Header.Add("x-real-ip", "95.82.232.185")
			resp, err := http.DefaultClient.Do(apiReq)
			require.NoError(t, err)

			if tt.ExpectedHTTPCode != 200 {
				require.Equal(t, tt.ExpectedHTTPCode, resp.StatusCode, "HTTP cods aren't equal")

				b, err := ioutil.ReadAll(resp.Body)
				require.NoError(t, err)

				resp.Body.Close()
				require.Equal(t, tt.ExpectedErrMsg, string(b))
			} else {
				require.Equal(t, http.StatusOK, resp.StatusCode, "HTTP code isn't 200")
				b, err := ioutil.ReadAll(resp.Body)
				require.NoError(t, err)
				resp.Body.Close()

				require.Equal(t, `{"status":"ok"}`, string(b))

				time.Sleep(200 * time.Millisecond)

				expectedAllBytes, err := ioutil.ReadFile(tt.ExpectedJSONPath)
				require.NoError(t, err)

				actualBytes := logging.InstanceMock.Data

				if expectedAllBytes[0] == '{' {
					require.Equal(t, 1, len(actualBytes))
					test.JSONBytesEqual(t, expectedAllBytes, actualBytes[0], "Logged facts aren't equal")
				} else {
					//array
					expectedEvents := []interface{}{}
					require.NoError(t, json.Unmarshal(expectedAllBytes, &expectedEvents))

					require.Equal(t, len(expectedEvents), len(actualBytes), "Logged facts count isn't equal with actual one")
					for i, expected := range expectedEvents {
						actualEvent := actualBytes[i]
						expectedBytes, err := json.Marshal(expected)
						require.NoError(t, err)
						test.JSONBytesEqual(t, expectedBytes, actualEvent, "Logged facts aren't equal")
					}
				}
			}
		})
	}
}

func TestSegmentAPIEndpoint(t *testing.T) {
	uuid.InitMock()
	binding.EnableDecoderUseNumber = true

	SetTestDefaultParams()
	tests := []struct {
		Name             string
		ReqURN           string
		ExpectedJSONPath string
	}{
		{
			"Segment API 2.0 ok",
			"/api/v1/segment",
			"test_data/segment_api_events_output_2.0.json",
		},
		{
			"Segment API compat ok",
			"/api/v1/segment/compat",
			"test_data/segment_api_events_output_compat.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			uuid.InitMock()
			binding.EnableDecoderUseNumber = true

			SetTestDefaultParams()

			testSuite := testsuite.NewSuiteBuilder(t).WithGeoDataMock().Build(t)
			defer testSuite.Close()

			sendSegmentRequests(t, "http://"+testSuite.HTTPAuthority()+tt.ReqURN)

			time.Sleep(1 * time.Second)

			actualBytes := logging.InstanceMock.Data

			expectedAllBytes, err := ioutil.ReadFile(tt.ExpectedJSONPath)
			require.NoError(t, err)

			expectedEvents := []interface{}{}
			require.NoError(t, json.Unmarshal(expectedAllBytes, &expectedEvents))

			require.Equal(t, len(expectedEvents), len(actualBytes), "Logged facts count isn't equal with actual one")
			for i, expected := range expectedEvents {
				actualEvent := actualBytes[i]
				expectedBytes, err := json.Marshal(expected)
				require.NoError(t, err)
				test.JSONBytesEqual(t, expectedBytes, actualEvent, "Logged facts aren't equal")
			}
		})
	}
}

func TestPostgresStreamInsert(t *testing.T) {
	configTemplate := `{"destinations": {
  			"test": {
        		"type": "postgres",
        		"mode": "stream",
				"only_tokens": ["s2stoken"],
        		"data_layout": {
          			"table_name_template": "events_without_pk"
				},
        		"datasource": {
          			"host": "%s",
					"port": %d,
          			"db": "%s",
          			"schema": "%s",
          			"username": "%s",
          			"password": "%s",
					"parameters": {
						"sslmode": "disable"
					}
        		}
      		}
    	}}`
	testPostgresStoreEvents(t, configTemplate, "/api/v1/s2s/event?token=s2stoken", 5, "events_without_pk", 5, false)
}

func TestPostgresStreamInsertWithPK(t *testing.T) {
	configTemplate := `{"destinations": {
  			"test": {
        		"type": "postgres",
        		"mode": "stream",
				"only_tokens": ["s2stoken"],
        		"data_layout": {
          			"table_name_template": "events_with_pk",
					"primary_key_fields": ["email"]
				},
        		"datasource": {
          			"host": "%s",
					"port": %d,
          			"db": "%s",
          			"schema": "%s",
          			"username": "%s",
          			"password": "%s",
					"parameters": {
						"sslmode": "disable"
					}
        		}
      		}
    	}}`
	testPostgresStoreEvents(t, configTemplate, "/api/v1/s2s/event?token=s2stoken", 5, "events_with_pk", 1, false)
}

func TestPostgresDryRun(t *testing.T) {
	configTemplate := `{"destinations": {
  			"test": {
        		"type": "postgres",
				"staged": true,
        		"mode": "stream",
				"only_tokens": ["s2stoken"],
				"data_layout": {
					"table_name_template": "dry_run"
				},
        		"datasource": {
          			"host": "%s",
					"port": %d,
          			"db": "%s",
          			"schema": "%s",
					"username": "%s",
          			"password": "%s",
					"parameters": {
						"sslmode": "disable"
					}
        		}
      		}
    	}}`
	testPostgresStoreEvents(t, configTemplate, "/api/v1/events/dry-run?token=s2stoken&destination_id=test", 1, "dry_run", 0, true)
}

func testPostgresStoreEvents(t *testing.T, pgDestinationConfigTemplate string, endpoint string, sendEventsCount int,
	tableName string, expectedEventsCount int, dryRun bool) {
	ctx := context.Background()
	container, err := test.NewPostgresContainer(ctx)
	if err != nil {
		t.Fatalf("failed to initialize container: %v", err)
	}
	defer container.Close()

	SetTestDefaultParams()
	destinationConfig := fmt.Sprintf(pgDestinationConfigTemplate, container.Host, container.Port, container.Database, container.Schema, container.Username, container.Password)

	testSuite := testsuite.NewSuiteBuilder(t).WithDestinationService(t, destinationConfig).Build(t)
	defer testSuite.Close()

	time.Sleep(500 * time.Millisecond)
	requestValue := []byte(`{"email": "test@domain.com"}`)
	for i := 0; i < sendEventsCount; i++ {
		apiReq, err := http.NewRequest("POST", "http://"+testSuite.HTTPAuthority()+endpoint, bytes.NewBuffer(requestValue))
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(apiReq)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, "HTTP code isn't 200")
		resp.Body.Close()
		time.Sleep(200 * time.Millisecond)
	}
	rows, err := container.CountRows(tableName)
	if !dryRun {
		require.NoError(t, err)
	}
	require.Equal(t, expectedEventsCount, rows)
}

//func TestClickhouseStreamInsert(t *testing.T) {
//	configTemplate := `{"destinations": {
//  			"test": {
//        		"type": "clickhouse",
//        		"mode": "stream",
//				"only_tokens": ["s2stoken"],
//        		"data_layout": {
//          			"table_name_template": "events_without_pk"
//				},
//        		"clickhouse": {
//          			"dsns": [%s],
//          			"db": "%s"
//        		}
//      		}
//    	}}`
//	testClickhouseStoreEvents(t, configTemplate, 5, "events_without_pk")
//}

func TestClickhouseStreamInsertWithMerge(t *testing.T) {
	configTemplate := `{"destinations": {
 			"test": {
       		"type": "clickhouse",
       		"mode": "stream",
				"only_tokens": ["s2stoken"],
       		"data_layout": {
         			"table_name_template": "events_with_pk"
				},
       		"clickhouse": {
         			"dsns": [%s],
         			"db": "%s",
					"engine": {
						"raw_statement": "ENGINE = ReplacingMergeTree(key) ORDER BY (key)"
					}
       		}
     		}
   	}}`
	testClickhouseStoreEvents(t, configTemplate, 5, "events_with_pk", 1)
}

func testClickhouseStoreEvents(t *testing.T, configTemplate string, sendEventsCount int, tableName string, expectedEventsCount int) {
	ctx := context.Background()
	container, err := test.NewClickhouseContainer(ctx)
	if err != nil {
		t.Fatalf("failed to initialize container: %v", err)
	}
	defer container.Close()
	telemetry.InitTest()
	viper.Set("log.path", "")
	//This test Uses deprecated config key (server.auth) for testing purposes.
	viper.Set("server.auth", `{"tokens":[{"id":"id1","server_secret":"s2stoken"}]}`)

	dsns := make([]string, len(container.Dsns))
	for i, dsn := range container.Dsns {
		dsns[i] = "\"" + dsn + "\""
	}
	destinationConfig := fmt.Sprintf(configTemplate, strings.Join(dsns, ","), container.Database)

	testSuite := testsuite.NewSuiteBuilder(t).WithDestinationService(t, destinationConfig).Build(t)
	defer testSuite.Close()

	requestValue := []byte(`{"email": "test@domain.com", "key": 1}`)
	for i := 0; i < sendEventsCount; i++ {
		apiReq, err := http.NewRequest("POST", "http://"+testSuite.HTTPAuthority()+"/api/v1/s2s/event?token=s2stoken", bytes.NewBuffer(requestValue))
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(apiReq)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, "HTTP code isn't 200")
		resp.Body.Close()
		time.Sleep(200 * time.Millisecond)
	}
	rows, err := container.CountRows(tableName)
	require.NoError(t, err)
	require.Equal(t, expectedEventsCount, rows)
}

func sendSegmentRequests(t *testing.T, endpoint string) {
	client, _ := analytics.NewWithConfig("s2stoken", analytics.Config{Endpoint: endpoint})
	analyticsTraits := analytics.NewTraits().SetFirstName("Ned").SetLastName("Stark").Set("role", "the Lord of Winterfell").SetEmail("ned@starkinc.com")
	analyticsContext := &analytics.Context{
		App: analytics.AppInfo{
			Name:      "app1",
			Version:   "1.0",
			Build:     "build",
			Namespace: "namespace",
		},
		Campaign: analytics.CampaignInfo{
			Name:    "my_cmp",
			Source:  "my_cmp_src",
			Medium:  "my_cmp_medium",
			Term:    "my_cmp_term",
			Content: "my_cmp_content",
		},
		Device: analytics.DeviceInfo{
			Id:            "device1",
			Manufacturer:  "Apple",
			Model:         "iPhone",
			Name:          "iPhone 12",
			Type:          "mobile",
			Version:       "12",
			AdvertisingID: "an_advertising_id",
		},
		Library: analytics.LibraryInfo{
			Name:    "lib1",
			Version: "1.0",
		},
		Location: analytics.LocationInfo{
			City:      "San Francisco",
			Country:   "US",
			Latitude:  1.0,
			Longitude: 2.0,
			Region:    "CA",
			Speed:     100,
		},
		Network: analytics.NetworkInfo{
			WIFI: true,
		},
		OS: analytics.OSInfo{
			Name:    "iOS",
			Version: "14.0",
		},
		Page: analytics.PageInfo{
			Hash:     "page_hash",
			Path:     "page_path",
			Referrer: "page_referrer",
			Search:   "page_search",
			Title:    "page_title",
			URL:      "page_url",
		},
		Referrer: analytics.ReferrerInfo{
			Type: "ref_type",
			Name: "ref_name",
			URL:  "ref_url",
			Link: "ref_link",
		},
		Screen: analytics.ScreenInfo{
			Density: 10,
			Width:   1024,
			Height:  768,
		},
		IP:        net.IPv4(1, 1, 1, 1),
		Locale:    "en-EU",
		Timezone:  "UTC",
		UserAgent: "Mozilla/5.0 (iPod; CPU iPhone OS 12_0 like macOS) AppleWebKit/602.1.50 (KHTML, like Gecko) Version/12.0 Mobile/14A5335d Safari/602.1.50",
		Traits:    analyticsTraits,
		Extra:     map[string]interface{}{"extra": true},
	}
	analyticsProperties := analytics.NewProperties().SetRevenue(10.0).SetCurrency("USD")
	integrations := analytics.NewIntegrations().EnableAll().Disable("Salesforce").Disable("Marketo")

	err := client.Enqueue(analytics.Track{
		MessageId:    "track1",
		AnonymousId:  "anonym1",
		Event:        "test_track",
		Timestamp:    time.Now(),
		Context:      analyticsContext,
		Properties:   analyticsProperties,
		Integrations: integrations,
	})
	require.NoError(t, err)
	err = client.Enqueue(analytics.Screen{
		MessageId:    "screen1",
		AnonymousId:  "anonym1",
		Name:         "home screen",
		Timestamp:    time.Now(),
		Context:      analyticsContext,
		Properties:   analyticsProperties,
		Integrations: integrations,
	})
	require.NoError(t, err)
	err = client.Enqueue(analytics.Alias{
		MessageId:    "alias1",
		PreviousId:   "previousId",
		UserId:       "user1",
		Timestamp:    time.Now(),
		Context:      analyticsContext,
		Integrations: integrations,
	})
	require.NoError(t, err)
	err = client.Enqueue(analytics.Group{
		MessageId:    "group1",
		AnonymousId:  "anonym1",
		UserId:       "user1",
		GroupId:      "group1",
		Timestamp:    time.Now(),
		Context:      analyticsContext,
		Traits:       analyticsTraits,
		Integrations: integrations,
	})
	require.NoError(t, err)
	err = client.Enqueue(analytics.Identify{
		MessageId:    "identify1",
		AnonymousId:  "anonym1",
		UserId:       "user1",
		Timestamp:    time.Now(),
		Context:      analyticsContext,
		Traits:       analyticsTraits,
		Integrations: integrations,
	})
	require.NoError(t, err)
	err = client.Enqueue(analytics.Page{
		MessageId:    "page1",
		AnonymousId:  "anonym1",
		Name:         "page",
		UserId:       "user1",
		Timestamp:    time.Now(),
		Context:      analyticsContext,
		Properties:   analyticsProperties,
		Integrations: integrations,
	})
	require.NoError(t, err)

	err = client.Close()
	require.NoError(t, err)
}

func sendSegmentRequestsWithoutLocationAndUA(t *testing.T, endpoint string) {
	client, _ := analytics.NewWithConfig("s2stoken", analytics.Config{Endpoint: endpoint})
	analyticsTraits := analytics.NewTraits().SetFirstName("Ned").SetLastName("Stark").Set("role", "the Lord of Winterfell").SetEmail("ned@starkinc.com")
	analyticsContext := &analytics.Context{
		App: analytics.AppInfo{
			Name:      "app1",
			Version:   "1.0",
			Build:     "build",
			Namespace: "namespace",
		},
		Campaign: analytics.CampaignInfo{
			Name:    "my_cmp",
			Source:  "my_cmp_src",
			Medium:  "my_cmp_medium",
			Term:    "my_cmp_term",
			Content: "my_cmp_content",
		},
		Library: analytics.LibraryInfo{
			Name:    "lib1",
			Version: "1.0",
		},
		Network: analytics.NetworkInfo{
			WIFI: true,
		},
		OS: analytics.OSInfo{
			Name:    "iOS",
			Version: "14.0",
		},
		Page: analytics.PageInfo{
			Hash:     "page_hash",
			Path:     "page_path",
			Referrer: "page_referrer",
			Search:   "page_search",
			Title:    "page_title",
			URL:      "page_url",
		},
		Referrer: analytics.ReferrerInfo{
			Type: "ref_type",
			Name: "ref_name",
			URL:  "ref_url",
			Link: "ref_link",
		},
		Screen: analytics.ScreenInfo{
			Density: 10,
			Width:   1024,
			Height:  768,
		},
		IP:        net.IPv4(10, 10, 10, 10),
		Locale:    "en-EU",
		Timezone:  "UTC",
		UserAgent: "Mozilla/5.0 (iPod; CPU iPhone OS 12_0 like macOS) AppleWebKit/602.1.50 (KHTML, like Gecko) Version/12.0 Mobile/14A5335d Safari/602.1.50",
		Traits:    analyticsTraits,
		Extra:     map[string]interface{}{"extra": true},
	}
	analyticsProperties := analytics.NewProperties().SetRevenue(10.0).SetCurrency("USD")
	integrations := analytics.NewIntegrations().EnableAll().Disable("Salesforce").Disable("Marketo")

	err := client.Enqueue(analytics.Track{
		MessageId:    "track1",
		AnonymousId:  "anonym1",
		Event:        "test_track",
		Timestamp:    time.Now(),
		Context:      analyticsContext,
		Properties:   analyticsProperties,
		Integrations: integrations,
	})
	require.NoError(t, err)
	err = client.Enqueue(analytics.Screen{
		MessageId:    "screen1",
		AnonymousId:  "anonym1",
		Name:         "home screen",
		Timestamp:    time.Now(),
		Context:      analyticsContext,
		Properties:   analyticsProperties,
		Integrations: integrations,
	})
	require.NoError(t, err)
	err = client.Enqueue(analytics.Alias{
		MessageId:    "alias1",
		PreviousId:   "previousId",
		UserId:       "user1",
		Timestamp:    time.Now(),
		Context:      analyticsContext,
		Integrations: integrations,
	})
	require.NoError(t, err)
	err = client.Enqueue(analytics.Group{
		MessageId:    "group1",
		AnonymousId:  "anonym1",
		UserId:       "user1",
		GroupId:      "group1",
		Timestamp:    time.Now(),
		Context:      analyticsContext,
		Traits:       analyticsTraits,
		Integrations: integrations,
	})
	require.NoError(t, err)
	err = client.Enqueue(analytics.Identify{
		MessageId:    "identify1",
		AnonymousId:  "anonym1",
		UserId:       "user1",
		Timestamp:    time.Now(),
		Context:      analyticsContext,
		Traits:       analyticsTraits,
		Integrations: integrations,
	})
	require.NoError(t, err)
	err = client.Enqueue(analytics.Page{
		MessageId:    "page1",
		AnonymousId:  "anonym1",
		Name:         "page",
		UserId:       "user1",
		Timestamp:    time.Now(),
		Context:      analyticsContext,
		Properties:   analyticsProperties,
		Integrations: integrations,
	})
	require.NoError(t, err)

	err = client.Close()
	require.NoError(t, err)
}
