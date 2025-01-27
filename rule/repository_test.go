/*
 * Copyright © 2017-2018 Aeneas Rekkas <aeneas+oss@aeneas.io>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * @author       Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @copyright  2017-2018 Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @license  	   Apache-2.0
 */

package rule

import (
	"context"
	"fmt"
	"testing"

	"github.com/bxcodec/faker"
	"github.com/pkg/errors"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/x/sqlcon/dockertest"
)

func TestMain(m *testing.M) {
	ex := dockertest.Register()
	code := m.Run()
	ex.Exit(code)
}

type validatorNoop struct {
	ret error
}

func (v *validatorNoop) Validate(*Rule) error {
	return v.ret
}

type mockRepositoryRegistry struct{ v validatorNoop }

func (r *mockRepositoryRegistry) RuleValidator() Validator {
	return &r.v
}

func TestRepository(t *testing.T) {
	for name, repo := range map[string]Repository{
		"memory": NewRepositoryMemory(
			new(mockRepositoryRegistry)),
	} {
		t.Run(fmt.Sprintf("repository=%s/case=valid rule", name), func(t *testing.T) {
			var rules []Rule
			for i := 0; i < 4; i++ {
				var rule Rule
				require.NoError(t, faker.FakeData(&rule))
				rules = append(rules, rule)
			}

			for _, expect := range rules {
				_, err := repo.Get(context.Background(), expect.ID)
				require.Error(t, err)
			}

			inserted := make([]Rule, len(rules))
			copy(inserted, rules)
			inserted = inserted[:len(inserted)-1] // insert all elements but the last
			require.NoError(t, repo.Set(context.Background(), inserted))

			for _, expect := range inserted {
				got, err := repo.Get(context.Background(), expect.ID)
				require.NoError(t, err)
				assert.EqualValues(t, expect, *got)
			}

			count, err := repo.Count(context.Background())
			require.NoError(t, err)
			assert.Equal(t, len(inserted), count)

			updated := make([]Rule, len(rules))
			copy(updated, rules)
			updated = append(updated[:len(updated)-2], updated[len(updated)-1]) // insert all elements (including last) except before last
			require.NoError(t, repo.Set(context.Background(), updated))

			count, err = repo.Count(context.Background())
			require.NoError(t, err)
			assert.Equal(t, len(updated), count)

			for _, expect := range updated {
				got, err := repo.Get(context.Background(), expect.ID)
				require.NoError(t, err)
				assert.EqualValues(t, expect, *got)
			}

			_, err = repo.Get(context.Background(), rules[len(rules)-2].ID) // check if before last still exists
			require.Error(t, err)

			count, err = repo.Count(context.Background())
			require.NoError(t, err)
			assert.Equal(t, len(rules)-1, count)
		})
	}

	for name, repo := range map[string]Repository{
		"memory": NewRepositoryMemory(&mockRepositoryRegistry{v: validatorNoop{ret: errors.New("")}}),
	} {
		t.Run(fmt.Sprintf("repository=%s/case=invalid rule", name), func(t *testing.T) {
			var rule Rule
			require.NoError(t, faker.FakeData(&rule))
			require.Error(t, repo.Set(context.Background(), []Rule{rule}))
		})
	}
}
