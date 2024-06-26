package rhsm2

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const consumerCreatedResponse = `{
  "created" : "2023-10-06T09:03:40+0000",
  "updated" : "2023-10-06T09:03:41+0000",
  "id" : "4028fcc68aef65d7018b043a70dc0ba9",
  "uuid" : "0b497970-760f-4623-943a-673c125f5b8e",
  "name" : "localhost",
  "username" : "admin",
  "entitlementStatus" : "disabled",
  "serviceLevel" : "",
  "role" : "",
  "usage" : "",
  "addOns" : [ ],
  "systemPurposeStatus" : "disabled",
  "releaseVer" : {
    "releaseVer" : null
  },
  "owner" : {
    "id" : "4028fcc68aef65d7018aef65ec030004",
    "key" : "donaldduck",
    "displayName" : "Donald Duck",
    "href" : "/owners/donaldduck",
    "contentAccessMode" : "org_environment"
  },
  "environment" : null,
  "entitlementCount" : 0,
  "facts" : {
    "system.certificate_version" : "3.2"
  },
  "lastCheckin" : null,
  "installedProducts" : [ {
    "created" : "2023-10-06T09:03:40+0000",
    "updated" : "2023-10-06T09:03:40+0000",
    "id" : "4028fcc68aef65d7018b043a70dc0bab",
    "productId" : "5050",
    "productName" : "Admin OS Premium Architecture Bits",
    "version" : "6.1",
    "arch" : "ppc64",
    "status" : null,
    "startDate" : null,
    "endDate" : null
  } ],
  "canActivate" : false,
  "capabilities" : [ ],
  "hypervisorId" : null,
  "contentTags" : [ ],
  "autoheal" : true,
  "annotations" : null,
  "contentAccessMode" : null,
  "type" : {
    "created" : null,
    "updated" : null,
    "id" : "1000",
    "label" : "system",
    "manifest" : false
  },
  "idCert" : {
    "created" : "2023-10-06T09:03:41+0000",
    "updated" : "2023-10-06T09:03:41+0000",
    "id" : "4028fcc68aef65d7018b043a757e0bae",
    "key" : "-----BEGIN PRIVATE KEY-----\n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\nBPZYa0neecp/0Xx1X+oOXCwQUweonw==\n-----END PRIVATE KEY-----\n",
    "cert" : "-----BEGIN CERTIFICATE-----\n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\n-----END CERTIFICATE-----\n",
    "serial" : {
      "created" : "2023-10-06T09:03:40+0000",
      "updated" : "2023-10-06T09:03:40+0000",
      "id" : 7499506966643821720,
      "serial" : 7499506966643821720,
      "expiration" : "2028-10-06T09:03:40+0000",
      "revoked" : false
    }
  },
  "guestIds" : [ ],
  "href" : "/consumers/0b497970-760f-4623-943a-673c125f5b8e",
  "activationKeys" : [ ],
  "serviceType" : null,
  "environments" : null
}`

const entitlementCertCreatedResponse = `[ {
  "created" : "2023-10-06T09:03:42+0000",
  "updated" : "2023-10-06T09:03:42+0000",
  "id" : "4028fcc68aef65d7018b043a76040baf",
  "key" : "-----BEGIN PRIVATE KEY-----\n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\nBPZYa0neecp/0Xx1X+oOXCwQUweonw==\n-----END PRIVATE KEY-----\n",
  "cert" : "-----BEGIN CERTIFICATE-----\n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\njp+17Fq/WtO/sMzhOW+4c7DYd6k=\n-----END CERTIFICATE-----\n-----BEGIN ENTITLEMENT DATA-----\neJzNXW1v2zYQ/iuBPocTRdmUnW/FsHXDMKAoigHrUBiKzaRCbMuz5S5BkP8+ipJs\nSSZFHiWHClAEMe9o6tEdeW+8vnrLdHs4btjeu/Pw/WQezSOMIoof0ISSEM0nYYxo\nFC4DMn2Y3s+Yd+sdjveH5T7ZZUm69e5evcPTkTPzeTK2zRbxcskOB062jTeMf/5z\n8fnNh+Lzt1sv3a/yr+OMWbzPOAnB/JsCjDD9gud3OLybkK98ArZdqQb5LLt9ujou\ns4N398+rl6w6VnDTWsKt94PtD2LxHv8j3i+/JxlbZsc9y2f7dlvNdJ46wLUfioMg\nIAFnzV52+Re8HDfnb4v/Y4d0w9IDOoRz/IxarJxwHd+ztRHlDw5Bmr+Zz2x181uc\n8Y92cfadf+Dnv/0s9U+T+GISvzUJqpbKtvH9mvFneYjXB1Y8dfW4j7vHxXGfL+kh\nWbM73/dZtvR3T4m/320QH/U/f/oTffz0Ef3xy99oGW9Xa7ZLtnzWDcviVZzFC/a8\nS/b88UOK8dutDLVA/IRa1FpMUrwuaQyQemRbtk+WNcRa06BqgS6wms/56zJDSJBK\ncalGrNAQzE4xaLwPrva57E60cDzP6IJOUJtZCpCaFqRqxTR+expULdgFeDOhYWSm\nxWu3WyJOLMXnPAbCg7P5nA1VC3D4/NTk+bkIdCBwGoViwEWiQoG6RWGqRSGJO0Co\nDYIwyPlOEEwdQUCCcBKQ8h/GCiS2jBsBqFwfKg98VOetIWNEfEIqY4cMlX+d0XpI\n0wIxMdminMyvT4KK5b4TaArMKJ1gJWotEUm36xcZHPLTScsCkzU6WZTz1EGkqFq/\nEkYvX4h3RQkMI255EQWEWfz4WBMiQVuDSzFsKlwFuy/YELmaMO3Zv0f+3KsF/z4B\n6ZcPH/Pdgv8i3jf5gU7nUG1ssWsV8pLeTidb87hTy3CGI8zN+pqvsWL3MnXkHyNB\nLdW8+ugJkl/T9OavNiIXepbz+oIXlQsZ4rkb5xVQKpqHVheN3dsX55erN940KXEv\njcEYpjEYstF0aAzG48HP1lHHxo46HsJRx2Nz1O1QU7jrKsq+qBUu+5hQ0zvvCZ1R\nI9AUhEAbaUalkI3Cwy8gs3PwFaApSXu79yVwrrx7CXCGPp6RrMkJ4b6fDLIR+IFR\nDz8wgviB0RB+YDSS4zMABVRM5ExFaRFokUmaq6CLBDWzMJwhZgMgJsXLVahOEueF\nJgZUAV67yK4mNVCGdp3mBsCugGDS7l0Vld2mJbgdusd5lAOKSzvg0k1lh0sRfhnH\nLo6n/ZzIKdCJnA7kRE4dRkOjIser8oFOB39LlKrPxd+nUVMwlulmk259EpVp23d0\na5pbTWiz1YRGW03Ya6sJ3YlEkVcxOKVUKRXjbMrlWVQmVJwdP63kgD4yUHphsOyA\nAZOFE6dMEXRql1cwXzNH0LYZ++zRgT4g2mFB2u/RgcNAaT1vRmfYIGOVG9Iq6ZpI\nRVLHATXCZcI4QdXq1dLIWa8pinXV6OEoU60QqtTZ1lGmYzGxbEqbFOfEJc0A/ovb\n84Nbw3MczuoHRy4RccIX/fsmfmQyrFbp8omLUJKPH5CYQoqXnA6imc0pfDEFKpc7\naK6r+U7AqtZiB5rl9grXmse1DWYeMFaZYo1hm8BwVeHiKhhc1ApaGOl6obmoIAQb\n6e4FxDAJoyx+Og/Cky2VaLjaa0sITLN3KgzqoxZZugoFV5m5vF6B8JWoanAe4id2\nDvXkxDUUpIMGINT5RMUEQeUSBj1FQgIoLiLdxUXErriIjK+4KI8i9zpPiT6A0RVb\n7nGiEodhjcZKiCiX0DtSz+3ENJGbZnI66IHrt6ZA1SJdnbxT8yr9qRSXasS6Sn86\noip9cdyotqPaoRq2hUF1+soJYSdx2Bab8jy63p4FgIz0q+oi+uRFh9LZ71NkNAmN\nnjs9dKMfap93uM0XCUTjbUu+nV+kCqHbFnYbhRB2U6iqmqnePdvsshf+LRe2UzO5\nI8hQnc5URATnIn+O0o4KdQUyHema0DBT1U56Nh8GmuwsM1Wh00zVTFyXg0QDOvMO\nM9OLc5JoAHUeDYAFRgRHFxgVgQ0cgtcpIKWmq+CoZH+b6tV8m1rp+DZtKLgOCE0+\nFhLUUL/axjA8sHF+reOwO62KhkxqhgYqGXJfMQQpEu0Qm9h2PxAFoWexGUEZKO2f\nKJSfITqOIRKF1H2isF1XZFnwrnCLFYQ9C94LZ9nZplUookWoXp/Sv1BKcKh+POl7\nqrdditphcAJfwwOvPFYm8WnnMe8J5nfTTmLo7pnE8axieJrzkYzAGTTxmzqtQ6hZ\nWPlNxKXf1M5CW94kUXhTCsKeN0lKD8uVCXFpedrdilDYnyrKvrciChvU2a2IygY1\nDDrJzU+A5SkPOrnv6gNuQiGYjI5/+7YTgnssAV1idQHEwJczzNxqr3+MKN9imH0y\nMbitbG0dVm5zU2GERQWEPiG1SVfJQ8L1SrBI8bkg0agae443uzX7iR/z/uUsvpgF\nlYu7BjhSlzcsN1CTahCY06vngVeOyB3f6hnUxrVgfj/Xt1+5toFv15EA7ZGBGktf\nC/ilrRY7DL8eF7la8zjM4AXRNIIHEgSX3pSoyCxtCcE+EtkyzXO2mAwOyGEcX8dZ\n0EvvD4yV/DSw8vl0WI2oHSWd2xa6NBpXaQl7Frrk3arGUugi7HrrBp5S0JSkA7Tv\ndJqzbGUpesdDlXmKIa/YKeOhdAzx0OIqq3H8QR6bubi0Co0/uM0DNsLs/fs6yqPs\nWpYh+jpORtDXsb6cIfwqOZx6nmH8qskY/KqiVaZprUZnFVNVqwHt3FDWakS9azXK\nZzGvMOt8nFqFGfCJahVmUe8KsyKCabyLykX6IlYJ3UUn4zEEuVXT76IB0PknEFnu\numbgsE8LsenTQkwcfXBlVrNPC3Hr1JdluYbdotSlufVRcGeoU2nuOPpBCUcL6pQq\nfCwr90rjlBYeluNeUNBqT+UFnTqBbbUndl7+ioGlbnI4GsN2pW7YaZ76BIVpdrpz\nR7EFo8hEn9EYRVc+ahMWVPxvPR3BZftdxW1Y8CQ5pi6NWnKa4xaXh8+S4+r68AkN\nQNG1YkupD9sVXWOn9Yt5N4wgnNUvHKeHbM+kPTuKEdGEQ17N0SIAdekoeEV7jgCV\nK7pW1nQ6RJsredLdgGmgNlfTMbS5qq+nR28hfRNCFYa2vYWctx+EgmXU1hLak7AF\nj9u2lvUWyeGMGsTIS2UTqRZNa2cQk42GJiGdydQ0QtWzaPX01ssXdQ11/fbGJ92l\nKZ/x9e3tfyGEy0w=\n-----END ENTITLEMENT DATA-----\n-----BEGIN RSA SIGNATURE-----\nBdVPVkPNxJAhKl8y/NHGVcx99V3kCmh+WMJSHXf+Fg9waeXWIZzAZJVos+6HIczL\n0jpvWqQhIVPmR8m8B2WLdyOpWQju3ICehlT1Rph0WIuLbqO7lxp4RihMzGadwfH9\ngw76u17PqG61k6gOKGqpPzTgD8ofMIkv7iRt/uhds87Pi6sSBih31WVOL+bYVuLq\nEwTTHU+18OnhpHJxlaOU7dRpWSsC99A3/HCYKXYS7dlHnPow65mkMo5HIa/aSSXq\n+1cqnvBvT2YzoSAVy/WqiQXqzUlj0ePNxVBDy89CXj4uLTRccOlDgeG2oCTdoXyJ\nc4i/O7gdECbbWhPgnOysd3QTtWG3M+Gvl2SDr+KUDZFVTY6M3ERZFdnaMIA7SMn1\nXtgfX1phVodZ6/5TvKVxltjOAE3ZgLnOsZbf8TXujHpSAgocmyghjdVWehnIOYZh\nbq7EIkLr2pv+aQiPtsU+OJeRDYHxprYeLTzY+L0OVFS2YAFfFkPPubKScsifeQXe\nnwXaVoPoVt/221BiSkp7vDRyw0s0RJ0636eVFPubx+T4zzItskxdyLF1I49XPo7R\ncsDv/wyDc8o3lLYoRTeh3YN6lIQG7a2uKCEbEjJ2FCe9ObnfyQbvosut+UHUiwcN\neNXQybEZSQA6/7ZFSVo0nwEls0+t1bB0DQJnu9JppLY=\n-----END RSA SIGNATURE-----\n",
  "serial" : {
    "created" : "2023-10-06T09:03:42+0000",
    "updated" : "2023-10-06T09:03:42+0000",
    "id" : 1454563328016773404,
    "serial" : 1454563328016773404,
    "expiration" : "2024-10-06T08:03:42+0000",
    "revoked" : false
  }
} ]`

// helperTestInstalledFiles check if all files were installed as expected
func helperTestInstalledFiles(t *testing.T, tempDirFilePath string) {
	// Check that consumer cert and key were installed
	expectedConsumerCertFilePath := filepath.Join(tempDirFilePath, "etc/pki/consumer/cert.pem")
	if _, err := os.Stat(expectedConsumerCertFilePath); err != nil {
		t.Fatalf("consumer cert file %s not installed", expectedConsumerCertFilePath)
	}
	expectedConsumerKeyFilePath := filepath.Join(tempDirFilePath, "etc/pki/consumer/key.pem")
	if _, err := os.Stat(expectedConsumerKeyFilePath); err != nil {
		t.Fatalf("consumer key file %s not installed", expectedConsumerKeyFilePath)
	}

	// Test that SCA entitlement certificate and key were installed
	expectedEntitlementCertFilePath := filepath.Join(tempDirFilePath, "etc/pki/entitlement/1454563328016773404.pem")
	if _, err := os.Stat(expectedEntitlementCertFilePath); err != nil {
		t.Fatalf("entitlement cert file %s not installed", expectedEntitlementCertFilePath)
	}
	expectedEntitlementKeyFilePath := filepath.Join(tempDirFilePath, "etc/pki/entitlement/1454563328016773404-key.pem")
	if _, err := os.Stat(expectedEntitlementKeyFilePath); err != nil {
		t.Fatalf("entitlement cert file %s not installed", expectedEntitlementKeyFilePath)
	}

	// Test that redhat.repo was generated
	expectedRepoFilePath := filepath.Join(tempDirFilePath, "etc/yum.repos.d/redhat.repo")
	if _, err := os.Stat(expectedRepoFilePath); err != nil {
		t.Fatalf("repo file %s not generated", expectedRepoFilePath)
	}
}

// TestRegisterUsernamePassword test the case, when system is successfully
// registered using username and password and no organization is provided
// Registration without any organization is the most typical way how system
// is registered
func TestRegisterUsernamePassword(t *testing.T) {
	t.Parallel()
	expectedConsumerUUID := "0b497970-760f-4623-943a-673c125f5b8e"
	handlerCounterConsumersPost := 0
	handlerCounterGetCertificates := 0
	var xCorrelationId string

	username := "admin"
	password := "admin"

	server := httptest.NewTLSServer(
		// It is expected that Register() method will call only
		// two REST API points
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Handler has to be a little bit more sophisticated in this
			// case, because we have to handle two types of REST API calls

			reqURL := req.URL.String()

			if req.Method == http.MethodPost && reqURL == "/consumers" {
				// Increase number of calls of this REST API endpoint
				handlerCounterConsumersPost += 1
				// Save correlation ID header for later checks
				xCorrelationId = req.Header.Get("X-Correlation-ID")

				// Return code 200
				rw.WriteHeader(200)
				// Add some headers specific for candlepin server
				rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
				// Return JSON document with consumer
				_, _ = rw.Write([]byte(consumerCreatedResponse))
			} else if req.Method == http.MethodGet && reqURL == "/consumers/"+expectedConsumerUUID+"/certificates" {
				// Increase number of calls of this REST API endpoint
				handlerCounterGetCertificates += 1

				// Check X-Correlation-ID
				currentXCorrId := req.Header.Get("X-Correlation-ID")
				if xCorrelationId != currentXCorrId {
					t.Fatalf("X-Correlation-ID: %s does not match ID: %s from the first HTTP request",
						currentXCorrId, xCorrelationId)
				}

				// Return code 200
				rw.WriteHeader(200)
				// Add some headers specific for candlepin server
				rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
				// Return JSON document with consumer
				_, _ = rw.Write([]byte(entitlementCertCreatedResponse))
			} else {
				t.Fatalf("unexpected REST API call: %s %s", req.Method, reqURL)
			}

		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, true, false, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// TODO: try to use secure connection
	rhsmClient.RHSMConf.Server.Insecure = true

	consumer, err := rhsmClient.RegisterUsernamePassword(&username, &password, nil, nil)
	if err != nil {
		t.Fatalf("registration failed: %s", err)
	}

	if consumer.Uuid != expectedConsumerUUID {
		t.Fatalf("expected consumer UUID: %s, got: %s", expectedConsumerUUID, consumer.Uuid)
	}

	if handlerCounterConsumersPost != 1 {
		t.Fatalf("REST API point POST /consumers not called once")
	}

	if handlerCounterGetCertificates != 1 {
		t.Fatalf("REST API point GET /consumers/%s/certificates not called once", expectedConsumerUUID)
	}

	helperTestInstalledFiles(t, tempDirFilePath)
}

// TestRegisterUsernamePasswordOrg test the case, when system is successfully
// registered using username and password
func TestRegisterUsernamePasswordOrg(t *testing.T) {
	t.Parallel()
	expectedConsumerUUID := "0b497970-760f-4623-943a-673c125f5b8e"
	handlerCounterConsumersPost := 0
	handlerCounterGetCertificates := 0
	var xCorrelationId string

	username := "admin"
	password := "admin"
	org := "donaldduck"
	var options = make(map[string]string)
	options["org"] = org

	server := httptest.NewTLSServer(
		// It is expected that Register() method will call only
		// two REST API points
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Handler has to be a little bit more sophisticated in this
			// case, because we have to handle two types of REST API calls

			reqURL := req.URL.String()

			if req.Method == http.MethodPost && reqURL == "/consumers?owner="+org {
				// Increase number of calls of this REST API endpoint
				handlerCounterConsumersPost += 1
				// Save correlation ID header for later checks
				xCorrelationId = req.Header.Get("X-Correlation-ID")

				// Return code 200
				rw.WriteHeader(200)
				// Add some headers specific for candlepin server
				rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
				// Return JSON document with consumer
				_, _ = rw.Write([]byte(consumerCreatedResponse))
			} else if req.Method == http.MethodGet && reqURL == "/consumers/"+expectedConsumerUUID+"/certificates" {
				// Increase number of calls of this REST API endpoint
				handlerCounterGetCertificates += 1

				// Check X-Correlation-ID
				currentXCorrId := req.Header.Get("X-Correlation-ID")
				if xCorrelationId != currentXCorrId {
					t.Fatalf("X-Correlation-ID: %s does not match ID: %s from the first HTTP request",
						currentXCorrId, xCorrelationId)
				}

				// Return code 200
				rw.WriteHeader(200)
				// Add some headers specific for candlepin server
				rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
				// Return JSON document with consumer
				_, _ = rw.Write([]byte(entitlementCertCreatedResponse))
			} else {
				t.Fatalf("unexpected REST API call: %s %s", req.Method, reqURL)
			}

		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, true, false, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// TODO: try to use secure connection
	rhsmClient.RHSMConf.Server.Insecure = true

	consumer, err := rhsmClient.RegisterUsernamePassword(&username, &password, &options, nil)
	if err != nil {
		t.Fatalf("registration failed: %s", err)
	}

	if consumer.Uuid != expectedConsumerUUID {
		t.Fatalf("expected consumer UUID: %s, got: %s", expectedConsumerUUID, consumer.Uuid)
	}

	if handlerCounterConsumersPost != 1 {
		t.Fatalf("REST API point POST /consumers?owner=%s not called once", org)
	}

	if handlerCounterGetCertificates != 1 {
		t.Fatalf("REST API point GET /consumers/%s/certificates not called once", expectedConsumerUUID)
	}

	helperTestInstalledFiles(t, tempDirFilePath)
}

const consumerWithEnvironmentsCreatedResponse = `{
  "created" : "2023-10-06T09:03:40+0000",
  "updated" : "2023-10-06T09:03:41+0000",
  "id" : "4028fcc68aef65d7018b043a70dc0ba9",
  "uuid" : "0b497970-760f-4623-943a-673c125f5b8e",
  "name" : "localhost",
  "username" : "admin",
  "entitlementStatus" : "disabled",
  "serviceLevel" : "",
  "role" : "",
  "usage" : "",
  "addOns" : [ ],
  "systemPurposeStatus" : "disabled",
  "releaseVer" : {
    "releaseVer" : null
  },
  "owner" : {
    "id" : "4028fcc68aef65d7018aef65ec030004",
    "key" : "donaldduck",
    "displayName" : "Donald Duck",
    "href" : "/owners/donaldduck",
    "contentAccessMode" : "org_environment"
  },
  "environment" : {
    "created": null,
    "updated": null,
    "id": null,
    "name": "env-name-1,env-name-2",
    "type": null,
    "description": null,
    "contentPrefix": null,
    "owner": null,
    "environmentContent": null
  },
  "entitlementCount" : 0,
  "facts" : {
    "system.certificate_version" : "3.2"
  },
  "lastCheckin" : null,
  "installedProducts" : [ {
    "created" : "2023-10-06T09:03:40+0000",
    "updated" : "2023-10-06T09:03:40+0000",
    "id" : "4028fcc68aef65d7018b043a70dc0bab",
    "productId" : "5050",
    "productName" : "Admin OS Premium Architecture Bits",
    "version" : "6.1",
    "arch" : "ppc64",
    "status" : null,
    "startDate" : null,
    "endDate" : null
  } ],
  "canActivate" : false,
  "capabilities" : [ ],
  "hypervisorId" : null,
  "contentTags" : [ ],
  "autoheal" : true,
  "annotations" : null,
  "contentAccessMode" : null,
  "type" : {
    "created" : null,
    "updated" : null,
    "id" : "1000",
    "label" : "system",
    "manifest" : false
  },
  "idCert" : {
    "created" : "2023-10-06T09:03:41+0000",
    "updated" : "2023-10-06T09:03:41+0000",
    "id" : "4028fcc68aef65d7018b043a757e0bae",
    "key" : "-----BEGIN PRIVATE KEY-----\n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\nBPZYa0neecp/0Xx1X+oOXCwQUweonw==\n-----END PRIVATE KEY-----\n",
    "cert" : "-----BEGIN CERTIFICATE-----\nMIIF9TCCA92gAwIBAgIIaBOZ2iFfTJgwDQYJKoZIhvcNAQELBQAwOzEaMBgGA1UE\nAwwRY2VudG9zOC1jYW5kbGVwaW4xCzAJBgNVBAYTAlVTMRAwDgYDVQQHDAdSYWxl\naWdoMB4XDTIzMTAwNjA4MDM0MFoXDTI4MTAwNjA5MDM0MFowRDETMBEGA1UECgwK\nZG9uYWxkZHVjazEtMCsGA1UEAwwkMGI0OTc5NzAtNzYwZi00NjIzLTk0M2EtNjcz\nYzEyNWY1YjhlMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAv4kcrkZI\n9iFgWwk055umWDjv0S1taE2GCQUBx/MifMQfdP/fJrw/3w6Fg56/uiNZbFEW6PLQ\ndmhTdB4rPzzfiKvRyBiU0wsta+W+P4fnxWC7dQdwatCUdnuwa58auv5CN3kcusXY\nKKY4pUV87wceLn0KmhilRSu0RUerEe2fvUJbKJNKEy41szo330xfozWkPWkLq6cR\nXjn+TssPS4MCmjRc3c6PX9LmJKIl59tQ1IUgQu15x/flQIbYIbZkWnSMin7OLvZL\n9Oa8v0OziCjEPcTBuBvN93wDOI3pgT1X8C9ISfeDOgfgix/38i4rqa0Ih3lioIPb\nxmdplbgqt+GeCTZRNfQkdq9EKn/rlkbSKR90TV2/l+mqTUsvEmS7UBBT3Zxci5sA\nSQfpYlkvTbijX49G91Y/p6a+81bC+seWketP74mLFLUqiJNIPWGMVMHUegSFlRyL\njemATKwCC49IeoXM3R1u+THM8NZc6pMwxnPM+Z1ZMAMa+Wtsmu49Lv0Mk9L5yr7Q\nNsxUIreWphxjNx5/su793w7LIy7vN5sw5jFvkhamOfvt5MlLYMxvNd1w8cew3BH7\nbFDWkfjKTbjmYjEJ2TyZRI9Xka1eFBqj4LuBOy9xJ/cXstQ3w3PHZzUic30716AC\nya5J0dUqjoOY6O5R7757xX2qUomWECeWC98CAwEAAaOB8zCB8DAOBgNVHQ8BAf8E\nBAMCBLAwEwYDVR0lBAwwCgYIKwYBBQUHAwIwCQYDVR0TBAIwADARBglghkgBhvhC\nAQEEBAMCBaAwHQYDVR0OBBYEFCFIYzC6T7p5jKHyWm2aFDOWxp0LMB8GA1UdIwQY\nMBaAFJE2hokZj5VQLw9nF1KgylvEX5akMGsGA1UdEQRkMGKkRjBEMRMwEQYDVQQK\nDApkb25hbGRkdWNrMS0wKwYDVQQDDCQwYjQ5Nzk3MC03NjBmLTQ2MjMtOTQzYS02\nNzNjMTI1ZjViOGWkGDAWMRQwEgYDVQQDDAt0aGlua3BhZC1wMTANBgkqhkiG9w0B\nAQsFAAOCAgEAczyOsDhoLMyee3JPt607b56ccO/MPrZ/moxx7IHDpLH/rkoAuRDK\nty7Ifs9rFjgzMN3NZehMrC1gvsAvkiEnQP0j4aE7p71gthITltVb9nHx9cYaE6Ox\nTMoHx0sLxDZFo8zY3wPcltFLHjj2JSRxwSAbr3HYczwNiwpbvdA7IACKAg+WJsQo\nUOaPK3XFRm6cBpEBM9sngeDxubwvvhcjWaxMaJ3Xk/y9j4udTr/sZdyZdGa/Clk+\n7ag+bgiCsNJ+XyWtbCQtmHvZxWB4i9MYE2VaCfxV8TKWoI4aMr8DhBqYPl9t86ER\nW4eXEGOT2oxHruWylVwNilScsKZfrc/CHemBAWXzA9n0QV2TYKR4EMi/breA29Eu\nH9cgyMyNeFqXFya376w92pUb+f4unt5ZfWLiOmPDjJYC9gn7iSwAu550Ez49Ra1q\niMn4sNmJ7uXynbw5OOKN/rmWlUvmZu5ddFr9CDG9wyDo1rD+ag8aSMSPkw/SDQor\nUC2VzCzM6PLh36B3UT9qDopH+xb+KWoqk6m+YiHne5xnxhYb7ros2qjUUaShXi1U\naXg/qhdWh9DTsTgrGotdV/oA+So3LNWvrerHtVq/xZeR9tAlWPepADO9gaBWu/r8\nHpLEj8XQpkU/MoxLBGiks29C2/T0uk3tbHsHspcDKlwDE/8cA0ZvJ3o=\n-----END CERTIFICATE-----\n",
    "serial" : {
      "created" : "2023-10-06T09:03:40+0000",
      "updated" : "2023-10-06T09:03:40+0000",
      "id" : 7499506966643821720,
      "serial" : 7499506966643821720,
      "expiration" : "2028-10-06T09:03:40+0000",
      "revoked" : false
    }
  },
  "guestIds" : [ ],
  "href" : "/consumers/0b497970-760f-4623-943a-673c125f5b8e",
  "activationKeys" : [ ],
  "serviceType" : null,
  "environments": [
    {
      "created": "2023-10-06T09:03:40+0000",
      "updated": "2023-10-06T09:03:40+0000",
      "id": "env-id-1",
      "name": "env-name-1",
      "type": null,
      "description": "Testing environment #1",
      "contentPrefix": null,
      "owner": {
        "id": "4028fcc68aef65d7018aef65ec030004",
        "key": "donaldduck",
        "displayName": "Donald Duck",
        "href": "/owners/donaldduck",
        "contentAccessMode": "org_environment"
      },
      "environmentContent": []
    },
    {
      "created": "2023-10-06T09:03:40+0000",
      "updated": "2023-10-06T09:03:40+0000",
      "id": "env-id-2",
      "name": "env-name-2",
      "type": null,
      "description": "Testing environment #2",
      "contentPrefix": null,
      "owner": {
        "id": "4028fcc68aef65d7018aef65ec030004",
        "key": "donaldduck",
        "displayName": "Donald Duck",
        "href": "/owners/donaldduck",
        "contentAccessMode": "org_environment"
      },
      "environmentContent": []
    }
  ]
}`

// TestRegisterUsernamePasswordOrgEnvironments test the case, when system is successfully
// registered using username and password and some environments
func TestRegisterUsernamePasswordOrgEnvironments(t *testing.T) {
	t.Parallel()
	expectedConsumerUUID := "0b497970-760f-4623-943a-673c125f5b8e"
	handlerCounterConsumersPost := 0
	handlerCounterGetCertificates := 0
	var xCorrelationId string

	username := "admin"
	password := "admin"
	org := "donaldduck"
	options := map[string]string{"environments": "env-id-1,env-id-2", "org": org, "foo": "bar"}

	server := httptest.NewTLSServer(
		// It is expected that Register() method will call only
		// two REST API points
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Handler has to be a little bit more sophisticated in this
			// case, because we have to handle two types of REST API calls

			reqURL := req.URL.String()

			if req.Method == http.MethodPost && reqURL == "/consumers?owner="+org {
				// Increase number of calls of this REST API endpoint
				handlerCounterConsumersPost += 1
				// Save correlation ID header for later checks
				xCorrelationId = req.Header.Get("X-Correlation-ID")

				// Return code 200
				rw.WriteHeader(200)
				// Add some headers specific for candlepin server
				rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
				// Return JSON document with consumer
				_, _ = rw.Write([]byte(consumerWithEnvironmentsCreatedResponse))
			} else if req.Method == http.MethodGet && reqURL == "/consumers/"+expectedConsumerUUID+"/certificates" {
				// Increase number of calls of this REST API endpoint
				handlerCounterGetCertificates += 1

				// Check X-Correlation-ID
				currentXCorrId := req.Header.Get("X-Correlation-ID")
				if xCorrelationId != currentXCorrId {
					t.Fatalf("X-Correlation-ID: %s does not match ID: %s from the first HTTP request",
						currentXCorrId, xCorrelationId)
				}

				// Return code 200
				rw.WriteHeader(200)
				// Add some headers specific for candlepin server
				rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
				// Return JSON document with consumer
				_, _ = rw.Write([]byte(entitlementCertCreatedResponse))
			} else {
				t.Fatalf("unexpected REST API call: %s %s", req.Method, reqURL)
			}

		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, true, false, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// TODO: try to use secure connection
	rhsmClient.RHSMConf.Server.Insecure = true

	consumer, err := rhsmClient.RegisterUsernamePassword(&username, &password, &options, nil)
	if err != nil {
		t.Fatalf("registration failed: %s", err)
	}

	if consumer.Uuid != expectedConsumerUUID {
		t.Fatalf("expected consumer UUID: %s, got: %s", expectedConsumerUUID, consumer.Uuid)
	}

	if len(consumer.Environments) != 2 {
		t.Fatalf("expected 2 environments, got: %d", len(consumer.Environments))
	}

	if handlerCounterConsumersPost != 1 {
		t.Fatalf("REST API point POST /consumers?owner=%s not called once", org)
	}

	if handlerCounterGetCertificates != 1 {
		t.Fatalf("REST API point GET /consumers/%s/certificates not called once", expectedConsumerUUID)
	}

	helperTestInstalledFiles(t, tempDirFilePath)
}

const invalidCredentials = `{
    "displayMessage": "Invalid Credentials",
    "requestUuid": "fd45b8ed-1b38-4166-9d9d-b87b061ff6fe"
}`

// TestFailedRegisterUsernamePasswordOrg test the case, when system fails during
// registration using username and password due to wrong password
func TestFailedRegisterUsernamePasswordOrg(t *testing.T) {
	t.Parallel()
	handlerCounterConsumersPost := 0

	username := "admin"
	password := "wrong password"
	org := "donaldduck"
	var options = make(map[string]string)
	options["org"] = org

	server := httptest.NewTLSServer(
		// It is expected that Register() method will call only
		// two REST API points
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Handler has to be a little bit more sophisticated in this
			// case, because we have to handle two types of REST API calls

			reqURL := req.URL.String()

			if req.Method == http.MethodPost && reqURL == "/consumers?owner="+org {
				// Increase number of calls of this REST API endpoint
				handlerCounterConsumersPost += 1

				// Return code 401
				rw.WriteHeader(401)
				// Add some headers specific for candlepin server
				rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
				// Return JSON document with consumer
				_, _ = rw.Write([]byte(invalidCredentials))
			} else {
				t.Fatalf("unexpected REST API call: %s %s", req.Method, reqURL)
			}

		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, true, false, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// TODO: try to use secure connection
	rhsmClient.RHSMConf.Server.Insecure = true

	consumer, err := rhsmClient.RegisterUsernamePassword(&username, &password, &options, nil)
	if err == nil {
		t.Fatalf("registration not failed, when wrong password provided")
	}

	if consumer != nil {
		t.Fatalf("expected consumer created despite wrong password provided")
	}

	if handlerCounterConsumersPost != 1 {
		t.Fatalf("REST API point POST /consumers?owner=%s not called once", org)
	}
}

// TestRegisterUsernamePasswordOrg test the case, when system is successfully
// registered using username and password, when there are no syspurpose file
func TestRegisterUsernamePasswordOrgNoSyspurpose(t *testing.T) {
	t.Parallel()
	expectedConsumerUUID := "0b497970-760f-4623-943a-673c125f5b8e"
	handlerCounterConsumersPost := 0
	handlerCounterGetCertificates := 0

	username := "admin"
	password := "admin"
	org := "donaldduck"
	var options = make(map[string]string)
	options["org"] = org

	server := httptest.NewTLSServer(
		// It is expected that Register() method will call only
		// two REST API points
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Handler has to be a little bit more sophisticated in this
			// case, because we have to handle two types of REST API calls

			reqURL := req.URL.String()

			if req.Method == http.MethodPost && reqURL == "/consumers?owner="+org {
				// Increase number of calls of this REST API endpoint
				handlerCounterConsumersPost += 1

				// Return code 200
				rw.WriteHeader(200)
				// Add some headers specific for candlepin server
				rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
				// Return JSON document with consumer
				_, _ = rw.Write([]byte(consumerCreatedResponse))
			} else if req.Method == http.MethodGet && reqURL == "/consumers/"+expectedConsumerUUID+"/certificates" {
				// Increase number of calls of this REST API endpoint
				handlerCounterGetCertificates += 1

				// Return code 200
				rw.WriteHeader(200)
				// Add some headers specific for candlepin server
				rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
				// Return JSON document with consumer
				_, _ = rw.Write([]byte(entitlementCertCreatedResponse))
			} else {
				t.Fatalf("unexpected REST API call: %s %s", req.Method, reqURL)
			}

		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, false, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// TODO: try to use secure connection
	rhsmClient.RHSMConf.Server.Insecure = true

	consumer, err := rhsmClient.RegisterUsernamePassword(&username, &password, &options, nil)
	if err != nil {
		t.Fatalf("registration failed: %s", err)
	}

	if consumer.Uuid != expectedConsumerUUID {
		t.Fatalf("expected consumer UUID: %s, got: %s", expectedConsumerUUID, consumer.Uuid)
	}

	if handlerCounterConsumersPost != 1 {
		t.Fatalf("REST API point POST /consumers?owner=%s not called once", org)
	}

	if handlerCounterGetCertificates != 1 {
		t.Fatalf("REST API point GET /consumers/%s/certificates not called once", expectedConsumerUUID)
	}

	helperTestInstalledFiles(t, tempDirFilePath)
}

const consumerCreatedResponseActivationKey = `{
    "created": "2023-11-03T15:49:34+0000",
    "updated": "2023-11-03T15:49:35+0000",
    "id": "4028fcc68b850d06018b95e01e4e0b5e",
    "uuid": "3d9f61ba-2776-43fe-8256-7a30918cdb96",
    "name": "thinkpad-p1",
    "username": null,
    "entitlementStatus": "disabled",
    "serviceLevel": "",
    "role": "",
    "usage": "",
    "addOns": [],
    "systemPurposeStatus": "disabled",
    "releaseVer": {
        "releaseVer": null
    },
    "owner": {
        "id": "4028fcc68b850d06018b850d1d860004",
        "key": "donaldduck",
        "displayName": "Donald Duck",
        "href": "/owners/donaldduck",
        "contentAccessMode": "org_environment"
    },
    "environment": null,
    "entitlementCount": 0,
    "facts": {
        "system.certificate_version": "3.2"
    },
    "lastCheckin": null,
    "installedProducts": [
        {
            "created": "2023-11-03T15:49:34+0000",
            "updated": "2023-11-03T15:49:34+0000",
            "id": "4028fcc68b850d06018b95e01e4e0b60",
            "productId": "900",
            "productName": "Multi-Attribute Limited Product (no content)",
            "version": "1.0",
            "arch": "x86_64",
            "status": null,
            "startDate": null,
            "endDate": null
        },
        {
            "created": "2023-11-03T15:49:34+0000",
            "updated": "2023-11-03T15:49:34+0000",
            "id": "4028fcc68b850d06018b95e01e4e0b62",
            "productId": "38072",
            "productName": "Fake OS Bits",
            "version": "1.0",
            "arch": "ALL",
            "status": null,
            "startDate": null,
            "endDate": null
        },
        {
            "created": "2023-11-03T15:49:34+0000",
            "updated": "2023-11-03T15:49:34+0000",
            "id": "4028fcc68b850d06018b95e01e4e0b61",
            "productId": "5050",
            "productName": "Admin OS Premium Architecture Bits",
            "version": "6.1",
            "arch": "ppc64",
            "status": null,
            "startDate": null,
            "endDate": null
        }
    ],
    "canActivate": false,
    "capabilities": [],
    "hypervisorId": null,
    "contentTags": [],
    "autoheal": true,
    "annotations": null,
    "contentAccessMode": null,
    "type": {
        "created": null,
        "updated": null,
        "id": "1000",
        "label": "system",
        "manifest": false
    },
    "idCert": {
        "created": "2023-11-03T15:49:35+0000",
        "updated": "2023-11-03T15:49:35+0000",
        "id": "4028fcc68b850d06018b95e020820b64",
        "key": "-----BEGIN PRIVATE KEY-----\nMIIJQwIBADANBgkqhkiG9w0BAQEFAASCCS0wggkpAgEAAoICAQDnTd6QHwkrEacd\nVP/06mcYo0W2ZjQw/GIZg0a2TqVqPu5H9WcfuQHBTcwRBCAh+7xZnnHK6ZLUYr0p\nBbMvHtXZxsYXIk/IrfbtkjbJ+UZiFuGlTe3hmh7FwAFHxV3iEfnW+dqmjdNgqhYx\noobaeCNzheyIt8TxQfCRQ+/cQJeF/PJzox+MpxBWsVCTsnDq19AlDEBQihO1kiF6\nLDXHUKevrdouBI3Z0lyWrUt6c88qvsKAncobda4PJ5mGj35UQFvRYit4zNhTA/6s\n4AncFaInlmDP1OtJg7RKBAqLT0fIgMZyxnoAc0pOmVcmVhboTKytFEAmc6r+R82M\nIams1spHCo4p2TqkOJqiOSjv3Qr3vln7L+Y5ubM5cckV+MF0eUNqAn7w/Mh14RZf\ncFHj/3MjZ9O49j29DZtjodr9c1jT+eQyr8WLihWZY+T/8xV64ZTQkwnAzP8sUqx2\nzA4LJR9GGw9VrChkEAti9xSFam72Dmp1PUFw1Ah3mfNH+nvYFH3EHVyx/rFcq3Ys\ngnqDr4nfdcoR06W4Qrfcl94CVzk+ndC3OjvQR7qDLVLZVNs4ThcMf5MWDu+UDW2s\nSPrDgVaJkYe3DM5DRjDMi//sMYslQuRkhp9zPodeJrX3BTmPw/S5oMP1jT+crs55\n1SPMNqx0pVq6usJTYm2s82pJYnFrjQIDAQABAoICADJcTPH6H+flrwn+4L+1bX76\nxYniFMj5mm5Nm87DrtZEH2NBdhN9F3m784tc55YeljK5AbOZSWO+n8rXt9yxjQQe\neDC/QrnBzrqJFC7DfGSM8NIp9nohMUQC+wRiX3+CRHBe9BFBXd11CuIBrfTRJX7Y\nkNhvkoyou7UVIptE4PjGSw+MW6knErKdZprRxi4LkQEP+szOU148E9fi2k6sK+Zf\nJtrQGQZs+8W+s2lU361B3V1dfYyfAafBCnc2tN2pKZScGDTz/MGOZBvFp7/oYebW\noft+M0BNUaqv2I/eUhXAm/SPbC6+61XToxrwsYogb5Q7jYARvRAiGAnDporANUf6\ncDTJbJBdgg4O1aGpq9HveJFGJa1DtYj8q/R6GxhNq3SdOJAnNVLaPRdVVAUapIPN\ndbyJNUxiD3LA60019XBLKaHFehQfRQjwDVBB7N3HiFRgE0cEvR16N4+4EIGSeRpg\nmMJaksAXU6kC7LMs/XE694fKn3IdzpZzi2I6fs6dMhhHeeiQQZbJpc0z4Cvr5fU4\nQvGd8h2gXigWqmcYMEev6zn9/zlLPKf0Yrw4+NsJg0Aa9qhCOzLcFKOhLqItkNG+\nxrzMbl0oHoY6xKZfA5u83LtbDIbzOGdttn4ij20nq9Zyc/I+lQHe4bMD5EMPgFVA\nrubTiFb/tK3HVNA1+8SNAoIBAQD5hybO7RtWegr1zDZDZmlwKlcjztDZuGMTISXj\nxadoTC7Jf1ANbhIr7njlxYYBR0DLCvE+7MnBmUmdrFd+ipA2Y+jE7rNbh5pCz4m0\nR1+BhoD6U7S3yppV+mZec68qaRORM5+mRSq0naFKP32pg0O1vuf/YL4BKiduQvUN\n5Y/2XOWs+8M1zS7rpwKr0tiTPQrOPnbsHdmA7PtrrVvjNbe7a0pAXAClER5YnjjH\nWFLQ0aVrx7CJCvCROAaIpUH+kaQ5Nig4I8C7URzyOsR34l16vgwwIdGKLBdZbpe5\nIRKkzan5fkPVEat9+Wb2+mC1/OI2hwQZ1aeTQISF8GkpKiuXAoIBAQDtTbaZrlSD\nI+HBkJDHa2cnJ8mH9Pk4oT7auBBpGUC31te+45SGHuDM0i+pQcdZMWKWAsiPb7Vk\n4KCIAvpBng1/1DxetHDaevm1oXfEGrz7AqILynZdbvJFOvSAL5Am8SwRM9+61qvf\ng8gx9aCUnQDnEnKYugW+D+pV0TIl7QGp0nnnxcdB/YK2qqyUjbq3y/6K8cf9+s7M\nfT2gMAtlgr+p/cOlHf1E3fm48ECJsEBINuUuQwFTGA5NBAu7KYtEY2q5v1izIaU/\n53K4vQsY8bugmXNXzXN+OOPWVyUmSK7xK4IlyPlP/dooER/BvQDdnPQjpseuxZ87\nWTHBu/qL/JZ7AoIBAQDpQEsot0AM0Dyak/BNhNge/5VBZbyZKOGEKkJO7p6UwlIi\nDR55z2CxLFRbBQ9E9qQ8qNfmaFYY4P+NomKPaVtxXGdkX3XNo66xsWsZ2HYQj/jV\nB5YxD/TBVjaHKECftmWSH92nAZTwDUGONFd1M70BkZkwgfBbClqdmX2VJfgy5ZfU\n4esSyeCIbC/1Cj1lNXR6Yq2r/iMpf11uxDjW5YoAakSYcOgxZRUm5ZBR1+BiTvcB\nD0OYd191gBereD9+hgoraNKj19f7wdD7JbAuGK4lZK3j+Fj1fk29TcIKTj4tTAsN\nq9ggvkm+knqLeHUa6W9UZVWSCMSCw5ad1aOX1VG1AoIBAQDVDtYoaw3z5E54NOzY\nL0UhMabOy5LLVnKRe3s4cXTa2ilAeSJLvQOGwhwKskVifmWOXBdoORq65LcdBtdb\n/KUQJSTCvu0oLAtxGPTn/D26fGWzgmb9gnRA1irZwFqDrDstgs6X+M+90Zr3yBoR\n9FjwHm8HUtV/DNkPjdChnobEy3WqM7ewNwWaLejfQ9Tl2mDCrUwra3kqvjOLVDeX\nP2oIHRAAesl/gHEKq6K2PKmqbKcd/wj8avfEDCLgOu+QSlZzSSpBQq8UjiBBX3HM\nRpV4Hu4hYhQUus1ND1cKZW20mKS0ehnlyPyKxr0lfCV/08CbFc3ozAQoC/ARik3s\nS69rAoIBADX3AX8KPA2KSffnjaLC284zJHBW6pstJXW6qxthIUtDuEKD7vNdDQsb\n8ono6B2WqEY+SdMcpqdUICWqTQcGmUhXTT53ddnvV4ItRyef/zgLE5Vg2bqWJgCc\nyizIuZ2aNFDrz3KSzBom0sNCOQtErNvXs9LQ2U2yuGqATOmqX7pkQxrH4r2eyb1J\nSso7gVYpf6JNOEhIeSkHvIc5d9XI+5TAH1xPPvyV8412+89By5HA2TH4KIr2Mvgl\nmOW7vnQNRs56sS2P2FLNvGtZK5W4sErEy127wX4zAAZTMYNVt5ayFobxPxwvtcAE\nEByL6OQlDGHNczBD5nZCXhA4PJMidjU=\n-----END PRIVATE KEY-----\n",
        "cert": "-----BEGIN CERTIFICATE-----\n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\n-----END CERTIFICATE-----\n",
        "serial": {
            "created": "2023-11-03T15:49:34+0000",
            "updated": "2023-11-03T15:49:34+0000",
            "id": 4491360281744523476,
            "serial": 4491360281744523476,
            "expiration": "2028-11-03T15:49:34+0000",
            "revoked": false
        }
    },
    "guestIds": [],
    "href": "/consumers/3d9f61ba-2776-43fe-8256-7a30918cdb96",
    "activationKeys": [
        {
            "activationKeyName": "awesome_os_pool",
            "activationKeyId": "4028fcc68b850d06018b850df87909c3"
        }
    ],
    "serviceType": null,
    "environments": null
}`

// TestRegisterActivationKeyOrg test the case, when system is successfully
// registered using activation key and organization ID
func TestRegisterActivationKeyOrg(t *testing.T) {
	t.Parallel()
	expectedConsumerUUID := "3d9f61ba-2776-43fe-8256-7a30918cdb96"
	handlerCounterConsumersPost := 0
	handlerCounterGetCertificates := 0
	handlerCounterGetContentOverriders := 0
	var xCorrelationId string

	orgId := "donaldduck"
	activationKey := "awesome_os_pool"

	server := httptest.NewTLSServer(
		// It is expected that Register() method will call only
		// two REST API points
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Handler has to be a little bit more sophisticated in this
			// case, because we have to handle two types of REST API calls

			reqURL := req.URL.String()

			if req.Method == http.MethodPost && reqURL == "/consumers?owner="+orgId+"&activation_keys="+activationKey {
				// Increase number of calls of this REST API endpoint
				handlerCounterConsumersPost += 1

				// Save X-Correlation-Id for later checks
				xCorrelationId = req.Header.Get("X-Correlation-ID")

				// Return code 200
				rw.WriteHeader(200)
				// Add some headers specific for candlepin server
				rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
				// Return JSON document with consumer
				_, _ = rw.Write([]byte(consumerCreatedResponseActivationKey))
			} else if req.Method == http.MethodGet && reqURL == "/consumers/"+expectedConsumerUUID+"/certificates" {
				// Increase number of calls of this REST API endpoint
				handlerCounterGetCertificates += 1

				// Check X-Correlation-ID to match ID from POST request
				currentXCorrId := req.Header.Get("X-Correlation-ID")
				if xCorrelationId != currentXCorrId {
					t.Fatalf("X-Correlation-ID %s does not match ID: %s from first HTTP request",
						currentXCorrId, xCorrelationId)
				}

				// Return code 200
				rw.WriteHeader(200)
				// Add some headers specific for candlepin server
				rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
				// Return JSON document with entitlement certificate and key
				_, _ = rw.Write([]byte(entitlementCertCreatedResponse))
			} else if req.Method == http.MethodGet && reqURL == "/consumers/"+expectedConsumerUUID+"/content_overrides" {
				handlerCounterGetContentOverriders += 1

				// Check X-Correlation-ID to match ID from POST request
				currentXCorrId := req.Header.Get("X-Correlation-ID")
				if xCorrelationId != currentXCorrId {
					t.Fatalf("X-Correlation-ID %s does not match ID: %s from first HTTP request",
						currentXCorrId, xCorrelationId)
				}

				// Return code 200
				rw.WriteHeader(200)
				// Add some headers specific for candlepin server
				rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
				// Return JSON document with empty list of content overrides
				_, _ = rw.Write([]byte("[]"))
			} else {
				t.Fatalf("unexpected REST API call: %s %s", req.Method, reqURL)
			}

		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, true, false, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// TODO: try to use secure connection
	rhsmClient.RHSMConf.Server.Insecure = true

	activationKeys := []string{activationKey}
	consumer, err := rhsmClient.RegisterOrgActivationKeys(&orgId, activationKeys, nil, nil)
	if err != nil {
		t.Fatalf("registration failed: %s", err)
	}

	if consumer.Uuid != expectedConsumerUUID {
		t.Fatalf("expected consumer UUID: %s, got: %s", expectedConsumerUUID, consumer.Uuid)
	}

	if handlerCounterConsumersPost != 1 {
		t.Fatalf("REST API point POST /consumers?owner=%s&activation_keys=%s not called once", orgId, activationKey)
	}

	if handlerCounterGetCertificates != 1 {
		t.Fatalf("REST API point GET /consumers/%s/certificates not called once", expectedConsumerUUID)
	}

	if handlerCounterGetContentOverriders != 1 {
		t.Fatalf("REST API point GET /consumers/%s/content_overrides not called once", expectedConsumerUUID)
	}

	helperTestInstalledFiles(t, tempDirFilePath)
}

// TestRegisterActivationKeyOrg test the case, when system is successfully
// registered using activation key and organization ID. This unit test also
// covers the case, when some content overrides is added to given activation key
func TestRegisterActivationKeyOrgContentOverride(t *testing.T) {
	t.Parallel()
	expectedConsumerUUID := "3d9f61ba-2776-43fe-8256-7a30918cdb96"
	handlerCounterConsumersPost := 0
	handlerCounterGetCertificates := 0
	handlerCounterGetContentOverriders := 0

	orgId := "donaldduck"
	activationKey := "awesome_os_pool"

	server := httptest.NewTLSServer(
		// It is expected that Register() method will call only
		// two REST API points
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Handler has to be a little bit more sophisticated in this
			// case, because we have to handle two types of REST API calls

			reqURL := req.URL.String()

			if req.Method == http.MethodPost && reqURL == "/consumers?owner="+orgId+"&activation_keys="+activationKey {
				// Increase number of calls of this REST API endpoint
				handlerCounterConsumersPost += 1

				// Return code 200
				rw.WriteHeader(200)
				// Add some headers specific for candlepin server
				rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
				// Return JSON document with consumer
				_, _ = rw.Write([]byte(consumerCreatedResponseActivationKey))
			} else if req.Method == http.MethodGet && reqURL == "/consumers/"+expectedConsumerUUID+"/certificates" {
				// Increase number of calls of this REST API endpoint
				handlerCounterGetCertificates += 1

				// Return code 200
				rw.WriteHeader(200)
				// Add some headers specific for candlepin server
				rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
				// Return JSON document with entitlement certificate and key
				_, _ = rw.Write([]byte(entitlementCertCreatedResponse))
			} else if req.Method == http.MethodGet && reqURL == "/consumers/"+expectedConsumerUUID+"/content_overrides" {
				handlerCounterGetContentOverriders += 1

				// Return code 200
				rw.WriteHeader(200)
				// Add some headers specific for candlepin server
				rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
				// Return JSON document with empty list of content overrides
				_, _ = rw.Write([]byte(contentOverridesList))
			} else {
				t.Fatalf("unexpected REST API call: %s %s", req.Method, reqURL)
			}

		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, true, false, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// TODO: try to use secure connection
	rhsmClient.RHSMConf.Server.Insecure = true

	activationKeys := []string{activationKey}
	consumer, err := rhsmClient.RegisterOrgActivationKeys(&orgId, activationKeys, nil, nil)
	if err != nil {
		t.Fatalf("registration failed: %s", err)
	}

	if consumer.Uuid != expectedConsumerUUID {
		t.Fatalf("expected consumer UUID: %s, got: %s", expectedConsumerUUID, consumer.Uuid)
	}

	if handlerCounterConsumersPost != 1 {
		t.Fatalf("REST API point POST /consumers?owner=%s&activation_keys=%s not called once", orgId, activationKey)
	}

	if handlerCounterGetCertificates != 1 {
		t.Fatalf("REST API point GET /consumers/%s/certificates not called once", expectedConsumerUUID)
	}

	if handlerCounterGetContentOverriders != 1 {
		t.Fatalf("REST API point GET /consumers/%s/content_overrides not called once", expectedConsumerUUID)
	}

	helperTestInstalledFiles(t, tempDirFilePath)
}

const consumerCreatedResponseTwoActivationKeys = `{
    "created": "2023-11-03T15:49:34+0000",
    "updated": "2023-11-03T15:49:35+0000",
    "id": "4028fcc68b850d06018b95e01e4e0b5e",
    "uuid": "3d9f61ba-2776-43fe-8256-7a30918cdb96",
    "name": "thinkpad-p1",
    "username": null,
    "entitlementStatus": "disabled",
    "serviceLevel": "",
    "role": "",
    "usage": "",
    "addOns": [],
    "systemPurposeStatus": "disabled",
    "releaseVer": {
        "releaseVer": null
    },
    "owner": {
        "id": "4028fcc68b850d06018b850d1d860004",
        "key": "donaldduck",
        "displayName": "Donald Duck",
        "href": "/owners/donaldduck",
        "contentAccessMode": "org_environment"
    },
    "environment": null,
    "entitlementCount": 0,
    "facts": {
        "system.certificate_version": "3.2"
    },
    "lastCheckin": null,
    "installedProducts": [
        {
            "created": "2023-11-03T15:49:34+0000",
            "updated": "2023-11-03T15:49:34+0000",
            "id": "4028fcc68b850d06018b95e01e4e0b60",
            "productId": "900",
            "productName": "Multi-Attribute Limited Product (no content)",
            "version": "1.0",
            "arch": "x86_64",
            "status": null,
            "startDate": null,
            "endDate": null
        },
        {
            "created": "2023-11-03T15:49:34+0000",
            "updated": "2023-11-03T15:49:34+0000",
            "id": "4028fcc68b850d06018b95e01e4e0b62",
            "productId": "38072",
            "productName": "Fake OS Bits",
            "version": "1.0",
            "arch": "ALL",
            "status": null,
            "startDate": null,
            "endDate": null
        },
        {
            "created": "2023-11-03T15:49:34+0000",
            "updated": "2023-11-03T15:49:34+0000",
            "id": "4028fcc68b850d06018b95e01e4e0b61",
            "productId": "5050",
            "productName": "Admin OS Premium Architecture Bits",
            "version": "6.1",
            "arch": "ppc64",
            "status": null,
            "startDate": null,
            "endDate": null
        }
    ],
    "canActivate": false,
    "capabilities": [],
    "hypervisorId": null,
    "contentTags": [],
    "autoheal": true,
    "annotations": null,
    "contentAccessMode": null,
    "type": {
        "created": null,
        "updated": null,
        "id": "1000",
        "label": "system",
        "manifest": false
    },
    "idCert": {
        "created": "2023-11-03T15:49:35+0000",
        "updated": "2023-11-03T15:49:35+0000",
        "id": "4028fcc68b850d06018b95e020820b64",
        "key": "-----BEGIN PRIVATE KEY-----\n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\nEByL6OQlDGHNczBD5nZCXhA4PJMidjU=\n-----END PRIVATE KEY-----\n",
        "cert": "-----BEGIN CERTIFICATE-----\n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\n-----END CERTIFICATE-----\n",
        "serial": {
            "created": "2023-11-03T15:49:34+0000",
            "updated": "2023-11-03T15:49:34+0000",
            "id": 4491360281744523476,
            "serial": 4491360281744523476,
            "expiration": "2028-11-03T15:49:34+0000",
            "revoked": false
        }
    },
    "guestIds": [],
    "href": "/consumers/3d9f61ba-2776-43fe-8256-7a30918cdb96",
    "activationKeys": [
        {
            "activationKeyName": "awesome_os_pool",
            "activationKeyId": "4028fcc68b850d06018b850df87909c3"
        },
        {
            "activationKeyName": "default_key",
            "activationKeyId": "4028fcc68baf3367018baf34623e09c2"
        }
    ],
    "serviceType": null,
    "environments": null
}`

// TestRegisterTwoActivationsKeyOrg test the case, when system is successfully
// registered using more than one activation key and organization ID. In this case
// two activation keys are used
func TestRegisterTwoActivationsKeyOrg(t *testing.T) {
	t.Parallel()
	expectedConsumerUUID := "3d9f61ba-2776-43fe-8256-7a30918cdb96"
	handlerCounterConsumersPost := 0
	handlerCounterGetCertificates := 0
	handlerCounterGetContentOverriders := 0

	orgId := "donaldduck"
	activationKeys := [2]string{"awesome_os_pool", "default_key"}

	server := httptest.NewTLSServer(
		// It is expected that Register() method will call only
		// two REST API points
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Handler has to be a little bit more sophisticated in this
			// case, because we have to handle two types of REST API calls

			reqURL := req.URL.String()

			var keys string
			for idx, activationKey := range activationKeys {
				keys += activationKey
				if idx < len(activationKeys)-1 {
					keys += ","
				}
			}

			if req.Method == http.MethodPost && reqURL == "/consumers?owner="+orgId+"&activation_keys="+keys {
				// Increase number of calls of this REST API endpoint
				handlerCounterConsumersPost += 1

				// Return code 200
				rw.WriteHeader(200)
				// Add some headers specific for candlepin server
				rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
				// Return JSON document with consumer
				_, _ = rw.Write([]byte(consumerCreatedResponseTwoActivationKeys))
			} else if req.Method == http.MethodGet && reqURL == "/consumers/"+expectedConsumerUUID+"/certificates" {
				// Increase number of calls of this REST API endpoint
				handlerCounterGetCertificates += 1

				// Return code 200
				rw.WriteHeader(200)
				// Add some headers specific for candlepin server
				rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
				// Return JSON document with consumer
				_, _ = rw.Write([]byte(entitlementCertCreatedResponse))
			} else if req.Method == http.MethodGet && reqURL == "/consumers/"+expectedConsumerUUID+"/content_overrides" {
				handlerCounterGetContentOverriders += 1

				// Return code 200
				rw.WriteHeader(200)
				// Add some headers specific for candlepin server
				rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
				// Return JSON document with empty list of content overrides
				_, _ = rw.Write([]byte("[]"))
			} else {
				t.Fatalf("unexpected REST API call: %s %s", req.Method, reqURL)
			}

		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, true, false, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// TODO: try to use secure connection
	rhsmClient.RHSMConf.Server.Insecure = true

	consumer, err := rhsmClient.RegisterOrgActivationKeys(&orgId, activationKeys[:], nil, nil)
	if err != nil {
		t.Fatalf("registration failed: %s", err)
	}

	if consumer.Uuid != expectedConsumerUUID {
		t.Fatalf("expected consumer UUID: %s, got: %s", expectedConsumerUUID, consumer.Uuid)
	}

	if handlerCounterConsumersPost != 1 {
		t.Fatalf("REST API point POST /consumers?owner=%s&activation_keys=%s not called once",
			orgId, activationKeys)
	}

	if handlerCounterGetCertificates != 1 {
		t.Fatalf("REST API point GET /consumers/%s/certificates not called once", expectedConsumerUUID)
	}

	if handlerCounterGetContentOverriders != 1 {
		t.Fatalf("REST API point GET /consumers/%s/content_overrides not called once", expectedConsumerUUID)
	}

	helperTestInstalledFiles(t, tempDirFilePath)
}
