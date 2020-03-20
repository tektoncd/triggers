package cel

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"reflect"
	"strings"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/interpreter/functions"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"k8s.io/client-go/kubernetes"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
)

func matchHeader(vals ...ref.Val) ref.Val {
	h, err := vals[0].ConvertToNative(reflect.TypeOf(http.Header{}))
	if err != nil {
		return types.NewErr("failed to convert to http.Header: %w", err)
	}

	key, ok := vals[1].(types.String)
	if !ok {
		return types.ValOrErr(key, "unexpected type '%v' passed to match", vals[1].Type())
	}

	val, ok := vals[2].(types.String)
	if !ok {
		return types.ValOrErr(val, "unexpected type '%v' passed to match", vals[2].Type())
	}

	return types.Bool(h.(http.Header).Get(string(key)) == string(val))
}

func truncateString(lhs, rhs ref.Val) ref.Val {
	str, ok := lhs.(types.String)
	if !ok {
		return types.ValOrErr(str, "unexpected type '%v' passed to truncate", lhs.Type())
	}

	n, ok := rhs.(types.Int)
	if !ok {
		return types.ValOrErr(n, "unexpected type '%v' passed to truncate", rhs.Type())
	}

	return types.String(str[:max(n, types.Int(len(str)))])
}

func splitString(lhs, rhs ref.Val) ref.Val {
	str, ok := lhs.(types.String)
	if !ok {
		return types.ValOrErr(str, "unexpected type '%v' passed to splitString", lhs.Type())
	}

	splitStr, ok := rhs.(types.String)
	if !ok {
		return types.ValOrErr(str, "unexpected type '%v' passed to splitString", lhs.Type())
	}

	r := types.NewRegistry()
	splitVals := strings.Split(string(str), string(splitStr))
	return types.NewStringList(r, splitVals)
}

func canonicalHeader(lhs, rhs ref.Val) ref.Val {
	h, err := lhs.ConvertToNative(reflect.TypeOf(http.Header{}))
	if err != nil {
		return types.NewErr("failed to convert to http.Header: %w", err)
	}

	key, ok := rhs.(types.String)
	if !ok {
		return types.ValOrErr(key, "unexpected type '%v' passed to canonical", rhs.Type())
	}

	return types.String(h.(http.Header).Get(string(key)))
}

func decodeB64String(val ref.Val) ref.Val {
	str, ok := val.(types.String)
	if !ok {
		return types.ValOrErr(str, "unexpected type '%v' passed to decodeB64", val.Type())
	}
	dec, err := base64.StdEncoding.DecodeString(str.Value().(string))
	if err != nil {
		return types.NewErr("failed to decode '%v' in decodeB64: %w", str, err)
	}
	return types.Bytes(dec)
}

func makeCompareSecret(defaultNS string, k kubernetes.Interface) functions.FunctionOp {
	return func(vals ...ref.Val) ref.Val {
		var ok bool
		compareString, ok := vals[0].(types.String)
		if !ok {
			return types.ValOrErr(compareString, "unexpected type '%v' passed to compareSecret", vals[0].Type())
		}
		paramCount := len(vals)

		var secretNS types.String
		if paramCount == 4 {
			secretNS, ok = vals[3].(types.String)
			if !ok {
				return types.ValOrErr(secretNS, "unexpected type '%v' passed to compareSecret", vals[1].Type())
			}
		} else {
			secretNS = types.String(defaultNS)
		}

		secretName, ok := vals[2].(types.String)
		if !ok {
			return types.ValOrErr(secretName, "unexpected type '%v' passed to match", vals[2].Type())
		}

		secretKey, ok := vals[1].(types.String)
		if !ok {
			return types.ValOrErr(secretKey, "unexpected type '%v' passed to match", vals[3].Type())
		}

		secretRef := &triggersv1.SecretRef{
			SecretKey:  string(secretKey),
			SecretName: string(secretName),
			Namespace:  string(secretNS),
		}
		secretToken, err := interceptors.GetSecretToken(k, secretRef, string(secretNS))
		if err != nil {
			return types.NewErr("failed to find secret '%#v' in compareSecret: %w", *secretRef, err)
		}
		return types.Bool(subtle.ConstantTimeCompare([]byte(secretToken), []byte(compareString)) == 1)
	}
}

func max(x, y types.Int) types.Int {
	switch x.Compare(y) {
	case types.IntNegOne:
		return x
	case types.IntOne:
		return y
	default:
		return x
	}
}
