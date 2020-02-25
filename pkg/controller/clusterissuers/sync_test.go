/*
Copyright 2019 The Jetstack cert-manager contributors.

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

package clusterissuers

import (
	"reflect"
	"runtime/debug"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgotesting "k8s.io/client-go/testing"

	"github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	testpkg "github.com/jetstack/cert-manager/pkg/controller/test"
)

func newFakeIssuerWithStatus(name string, status v1alpha2.IssuerStatus) *v1alpha2.ClusterIssuer {
	return &v1alpha2.ClusterIssuer{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: status,
	}
}

func TestSync(t *testing.T) {

}

func TestUpdateIssuerStatus(t *testing.T) {
	b := &testpkg.Builder{
		T: t,
	}
	b.Init()
	defer b.Stop()

	c := &controller{}
	c.Register(b.Context)
	b.Start()

	fakeClient := b.FakeCMClient()
	assertNumberOfActions(t, fatalf, filter(fakeClient.Actions()), 0)

	originalIssuer := newFakeIssuerWithStatus("test", v1alpha2.IssuerStatus{})

	issuer, err := fakeClient.CertmanagerV1alpha2().ClusterIssuers().Create(originalIssuer)
	assertErrIsNil(t, fatalf, err)

	assertNumberOfActions(t, fatalf, filter(fakeClient.Actions()), 1)

	newStatus := v1alpha2.IssuerStatus{
		Conditions: []v1alpha2.IssuerCondition{
			{
				Type:   v1alpha2.IssuerConditionReady,
				Status: cmmeta.ConditionTrue,
			},
		},
	}

	issuerCopy := issuer.DeepCopy()
	issuerCopy.Status = newStatus
	_, err = c.updateIssuerStatus(issuer, issuerCopy)
	assertErrIsNil(t, fatalf, err)

	actions := filter(fakeClient.Actions())
	assertNumberOfActions(t, fatalf, actions, 2)

	action := actions[1]
	updateAction := assertIsUpdateAction(t, errorf, action)

	obj := updateAction.GetObject()
	issuer = assertIsClusterIssuer(t, errorf, obj)

	assertDeepEqual(t, errorf, newStatus, issuer.Status)
}

func assertIsUpdateAction(t *testing.T, f failfFunc, action clientgotesting.Action) clientgotesting.UpdateAction {
	updateAction, ok := action.(clientgotesting.UpdateAction)
	if !ok {
		f(t, "action %#v does not implement interface UpdateAction")
	}
	return updateAction
}

func assertNumberOfActions(t *testing.T, f failfFunc, actions []clientgotesting.Action, number int) {
	if len(actions) != number {
		f(t, "expected %d actions, but got %d", number, len(actions))
	}
}

func assertErrIsNil(t *testing.T, f failfFunc, err error) {
	if err != nil {
		f(t, err.Error())
	}
}

func assertIsClusterIssuer(t *testing.T, f failfFunc, obj runtime.Object) *v1alpha2.ClusterIssuer {
	issuer, ok := obj.(*v1alpha2.ClusterIssuer)
	if !ok {
		f(t, "expected runtime.Object to be of type *v1alpha2.Issuer, but it was %#v", obj)
	}
	return issuer
}

func assertDeepEqual(t *testing.T, f failfFunc, left, right interface{}) {
	if !reflect.DeepEqual(left, right) {
		f(t, "object '%#v' does not equal '%#v'", left, right)
	}
}

func filter(in []clientgotesting.Action) []clientgotesting.Action {
	var out []clientgotesting.Action
	for _, i := range in {
		if i.GetVerb() != "list" && i.GetVerb() != "watch" {
			out = append(out, i)
		}
	}
	return out
}

// failfFunc is a type that defines the common signatures of T.Fatalf and
// T.Errorf.
type failfFunc func(t *testing.T, msg string, args ...interface{})

func fatalf(t *testing.T, msg string, args ...interface{}) {
	t.Log(string(debug.Stack()))
	t.Fatalf(msg, args...)
}

func errorf(t *testing.T, msg string, args ...interface{}) {
	t.Log(string(debug.Stack()))
	t.Errorf(msg, args...)
}