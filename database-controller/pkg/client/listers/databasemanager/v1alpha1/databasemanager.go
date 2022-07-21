/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/BlackBN/KubenetesDemo/database-controller/pkg/apis/databasemanager/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// DatabaseManagerLister helps list DatabaseManagers.
// All objects returned here must be treated as read-only.
type DatabaseManagerLister interface {
	// List lists all DatabaseManagers in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.DatabaseManager, err error)
	// DatabaseManagers returns an object that can list and get DatabaseManagers.
	DatabaseManagers(namespace string) DatabaseManagerNamespaceLister
	DatabaseManagerListerExpansion
}

// databaseManagerLister implements the DatabaseManagerLister interface.
type databaseManagerLister struct {
	indexer cache.Indexer
}

// NewDatabaseManagerLister returns a new DatabaseManagerLister.
func NewDatabaseManagerLister(indexer cache.Indexer) DatabaseManagerLister {
	return &databaseManagerLister{indexer: indexer}
}

// List lists all DatabaseManagers in the indexer.
func (s *databaseManagerLister) List(selector labels.Selector) (ret []*v1alpha1.DatabaseManager, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.DatabaseManager))
	})
	return ret, err
}

// DatabaseManagers returns an object that can list and get DatabaseManagers.
func (s *databaseManagerLister) DatabaseManagers(namespace string) DatabaseManagerNamespaceLister {
	return databaseManagerNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// DatabaseManagerNamespaceLister helps list and get DatabaseManagers.
// All objects returned here must be treated as read-only.
type DatabaseManagerNamespaceLister interface {
	// List lists all DatabaseManagers in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.DatabaseManager, err error)
	// Get retrieves the DatabaseManager from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.DatabaseManager, error)
	DatabaseManagerNamespaceListerExpansion
}

// databaseManagerNamespaceLister implements the DatabaseManagerNamespaceLister
// interface.
type databaseManagerNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all DatabaseManagers in the indexer for a given namespace.
func (s databaseManagerNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.DatabaseManager, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.DatabaseManager))
	})
	return ret, err
}

// Get retrieves the DatabaseManager from the indexer for a given namespace and name.
func (s databaseManagerNamespaceLister) Get(name string) (*v1alpha1.DatabaseManager, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("databasemanager"), name)
	}
	return obj.(*v1alpha1.DatabaseManager), nil
}
