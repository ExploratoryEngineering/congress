package storagetest

//
//Copyright 2018 Telenor Digital AS
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
import (
	"testing"

	"github.com/ExploratoryEngineering/congress/model"
	"github.com/ExploratoryEngineering/congress/storage"
)

// TokenStorageTest tests a storage.TokenStorage instance
func testTokenStorage(tokenStore storage.TokenStorage, userID model.UserID, t *testing.T) {
	defer tokenStore.Close()

	token1, _ := model.NewAPIToken(userID, "/", false)
	token2, _ := model.NewAPIToken(userID, "/some", true)
	token3, _ := model.NewAPIToken(userID, "/thing", false)

	// Make sure delete fails the first time
	if err := tokenStore.Delete(token1.Token, userID); err != storage.ErrNotFound {
		t.Fatal("Expected error when deleting token but got none")
	}

	// Add tokens to the store
	if err := tokenStore.Put(token1, userID); err != nil {
		t.Fatal("Got error storing token 1: ", err)
	}
	if err := tokenStore.Put(token2, userID); err != nil {
		t.Fatal("Got error storing token 2: ", err)
	}
	if err := tokenStore.Put(token3, userID); err != nil {
		t.Fatal("Got error storing token 3: ", err)
	}

	// Adding the same token twice returns error
	if err := tokenStore.Put(token1, userID); err != storage.ErrAlreadyExists {
		t.Fatal("Expected ErrAlreadyExists but got ", err)
	}

	// Should not be able to delete another user's token
	if err := tokenStore.Delete(token1.Token, model.UserID("foo")); err != storage.ErrNotFound {
		t.Fatal("Should not be able to remove another user's token")
	}

	// Retrieving a token that doesn't exist returns error
	if _, err := tokenStore.Get("some random value that doesn't exist"); err != storage.ErrNotFound {
		t.Fatal("Expected ErrNotFound but got ", err)
	}
	// Retrieve the tokens, one by one and compare with the original
	getOne := func(token model.APIToken) {
		tmp, err := tokenStore.Get(token.Token)
		if err != nil {
			t.Fatal("Got error retrieving token: ", err)
		}
		if !tmp.Equals(token) {
			t.Fatalf("Did not get the same token back: %v != %v", tmp, token)
		}
	}

	getOne(token1)
	getOne(token2)
	getOne(token3)

	// Retrieve the list. All three should be in it
	foundTokens := 0
	list, err := tokenStore.GetList(userID)
	if err != nil {
		t.Fatal("Got error retrieving list: ", err)
	}

	for v := range list {
		if v.Equals(token1) {
			foundTokens++
		}
		if v.Equals(token2) {
			foundTokens += 2
		}
		if v.Equals(token3) {
			foundTokens += 3
		}
	}

	if foundTokens != 6 {
		t.Fatalf("Didn't find all of the tokens. Expected 1 + 2 + 3 but got %d", foundTokens)
	}

	token1.Tags.SetTag("name", "value")
	if err := tokenStore.Update(token1, userID); err != nil {
		t.Fatal("Got error updating token 1: ", err)
	}
	// Remove all of the tokens. Should succeed the first time.
	if err := tokenStore.Delete(token1.Token, userID); err != nil {
		t.Fatal("Got error deleting token 1: ", err)
	}
	if err := tokenStore.Delete(token2.Token, userID); err != nil {
		t.Fatal("Got error deleting token 2: ", err)
	}
	if err := tokenStore.Delete(token3.Token, userID); err != nil {
		t.Fatal("Got error deleting token 3: ", err)
	}
	// Another delete will fail
	if err := tokenStore.Delete(token1.Token, userID); err != storage.ErrNotFound {
		t.Fatal("Expected ErrNotFound when deleting token but got none")
	}
}
