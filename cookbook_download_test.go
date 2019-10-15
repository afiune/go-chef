//
//  Author:: Salim Afiune <afiune@chef.io>
//

package chef

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

const emptyCookbookResponseFile = "test/empty_cookbook.json"

func TestDownloadCookbookThatDoesNotExist(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/cookbooks/foo/2.1.0", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", 404)
	})

	err := client.Cookbooks.DownloadCookbook("foo", "2.1.0")
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "404")
	}
}

func TestDownloadCookbookCorrectsLatestVersion(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/cookbooks/foo/_latest", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", 404)
	})

	err := client.Cookbooks.DownloadCookbook("foo", "")
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "404")
	}

	err = client.Cookbooks.DownloadCookbook("foo", "latest")
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "404")
	}

	err = client.Cookbooks.DownloadCookbook("foo", "_latest")
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "404")
	}
}

func TestDownloadCookbookEmptyWithVersion(t *testing.T) {
	setup()
	defer teardown()

	cbookResp, err := ioutil.ReadFile(emptyCookbookResponseFile)
	if err != nil {
		t.Error(err)
	}

	mux.HandleFunc("/cookbooks/foo/0.2.0", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, string(cbookResp))
	})

	err = client.Cookbooks.DownloadCookbook("foo", "0.2.0")
	assert.Nil(t, err)
}

func TestDownloadCookbookAt(t *testing.T) {
	setup()
	defer teardown()

	mockedCookbookResponseFile := `
{
  "version": "0.2.1",
  "name": "foo-0.2.1",
  "cookbook_name": "foo",
  "frozen?": false,
  "chef_type": "cookbook_version",
  "json_class": "Chef::CookbookVersion",
  "attributes": [],
  "definitions": [],
  "files": [],
  "libraries": [],
  "providers": [],
  "recipes": [
    {
      "name": "default.rb",
      "path": "recipes/default.rb",
      "checksum": "320sdk2w38020827kdlsdkasbd5454b6",
      "specificity": "default",
      "url": "` + server.URL + `/bookshelf/foo/default_rb"
    }
  ],
  "resources": [],
  "root_files": [
    {
      "name": "metadata.rb",
      "path": "metadata.rb",
      "checksum": "14963c5b685f3a15ea90ae51bd5454b6",
      "specificity": "default",
      "url": "` + server.URL + `/bookshelf/foo/metadata_rb"
    }
  ],
  "templates": [],
  "metadata": {},
  "access": {}
}
`

	tempDir, err := ioutil.TempDir("", "foo-cookbook")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(tempDir) // clean up

	mux.HandleFunc("/cookbooks/foo/0.2.1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, string(mockedCookbookResponseFile))
	})
	mux.HandleFunc("/bookshelf/foo/metadata_rb", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "name 'foo'")
	})
	mux.HandleFunc("/bookshelf/foo/default_rb", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "log 'this is a resource'")
	})

	err = client.Cookbooks.DownloadCookbookAt("foo", "0.2.1", tempDir)
	assert.Nil(t, err)

	var (
		cookbookPath = path.Join(tempDir, "foo-0.2.1")
		metadataPath = path.Join(cookbookPath, "metadata.rb")
		recipesPath  = path.Join(cookbookPath, "recipes")
		defaultPath  = path.Join(recipesPath, "default.rb")
	)
	assert.DirExistsf(t, cookbookPath, "the cookbook directory should exist")
	assert.DirExistsf(t, recipesPath, "the recipes directory should exist")
	if assert.FileExistsf(t, metadataPath, "a metadata.rb file should exist") {
		metadataBytes, err := ioutil.ReadFile(metadataPath)
		assert.Nil(t, err)
		assert.Equal(t, "name 'foo'", string(metadataBytes))
	}
	if assert.FileExistsf(t, defaultPath, "the default.rb recipes should exist") {
		recipeBytes, err := ioutil.ReadFile(defaultPath)
		assert.Nil(t, err)
		assert.Equal(t, "log 'this is a resource'", string(recipeBytes))
	}
}
