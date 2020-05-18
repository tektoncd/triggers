package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"reflect"
	"regexp"
	"strings"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/plugin/ochttp/propagation/b3"
	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"knative.dev/pkg/logging"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/template"
)

var (
	ml *zap.SugaredLogger

	traceClient *http.Client
)

func RunAdapter(ctx context.Context) {
	ce, err := cloudevents.NewDefaultClient()
	if err != nil {
		log.Fatalf("failed to create CloudEvent client, %s", err)
	}

	ml = logging.FromContext(ctx) // init local
	traceClient = &http.Client{Transport: &ochttp.Transport{}}

	log.Fatal(ce.StartReceiver(ctx, receive))
}

func receive(ctx context.Context, event cloudevents.Event) {
	ml.Debugf("Received CloudEvent,\n%s", event)

	l := ml.With("eventId", event.ID())
	l.Infof("do event. ")

	de := DataEvent{
		Context: event.Context,
	}
	err := json.Unmarshal(event.Data(), &de.Data)
	if err != nil {
		l.Errorf("load event data failed %v. ", err)
		return
	}

	if l.Desugar().Core().Enabled(zap.DebugLevel) {
		m, _ := json.Marshal(de)
		l.Debug("do data: ", string(m))
	}

	for _, conf := range CurrentConfig {
		ok, listeners := GetFilterListeners(l, &de, &conf)
		if !ok {
			continue
		}
		if len(listeners) == 0 {
			l.Infof("filter %v matched, but has no listener, event do end. ", conf.Filters)
			return
		}

		DoListenerSend(ctx, l, listeners, &de)
		return
	}

}

func GetFilterListeners(l *zap.SugaredLogger, de *DataEvent, conf *AdapterConfig) (bool, []ListenerInfo) {
	var res []ListenerInfo

	filter := true
	for _, v := range conf.Filters {
		value, err := template.ParseJSONPath(de, v.Key)
		if err != nil {
			l.Warnf("parse key:%s failed %v. ", v.Key, err)
			filter = false
			continue
		}
		if !v.IsMatch(value) {
			filter = false
		}
	}
	if filter {
		res = append(res, conf.DestListeners...)
	} else {
		l.Debugf("filter match failed, skip %#v. ", conf)
		return false, nil
	}

	return true, res
}

var reg = regexp.MustCompile(`\[|"|\]|\\`)

var (
	TraceID  = strings.ReplaceAll(b3.TraceIDHeader, "-", "")
	SpanID   = strings.ReplaceAll(b3.SpanIDHeader, "-", "")
	SampleID = strings.ReplaceAll(b3.SampledHeader, "-", "")
)

func TrimID(src string) string {
	return reg.ReplaceAllString(src, "")
}

func DoListenerSend(ctx context.Context, l *zap.SugaredLogger, listeners []ListenerInfo, de *DataEvent) {
	var err error
	var parentSpanCtx trace.SpanContext

	//parentSpan := trace.FromContext(ctx)
	//if parentSpan == nil {
	span := trace.SpanContext{}
	id, err := de.Context.GetExtension(TraceID)
	if err == nil {
		tid, ok := b3.ParseTraceID(TrimID(id.(string)))
		if ok {
			span.TraceID = tid
		} else {
			l.Warnf("parse %s failed %v. ", TraceID, reflect.TypeOf(id), id)
		}
	} else {
		l.Warnf("get invalid extension %s: %v. ", TraceID, err)
	}
	id, err = de.Context.GetExtension(SpanID)
	if err == nil {
		sid, ok := b3.ParseSpanID(TrimID(id.(string)))
		if ok {
			span.SpanID = sid
		} else {
			l.Warnf("parse %s failed %v. ", SpanID, id)
		}
	} else {
		l.Warnf("get invalid extension %s: %v. ", SpanID, err)
	}
	id, err = de.Context.GetExtension(SampleID)
	if err == nil {
		sample, ok := b3.ParseSampled(TrimID(id.(string)))
		if ok {
			span.TraceOptions = sample
		} else {
			l.Warnf("parse %s failed %v. ", SampleID, id)
		}
	} else {
		l.Warnf("get invalid extension %s: %v. ", SampleID, err)
	}
	parentSpanCtx = span
	l.Infof("get span from event extensions. ")
	//} else {
	//	parentSpanCtx = parentSpan.SpanContext()
	//}
	l.Infof("get parent span: %v. ", parentSpanCtx)

	for _, listener := range listeners {
		// resole bindings
		resBindings := make([]v1alpha1.Param, len(listener.Bindings))
		for i, p := range listener.Bindings {
			resBindings[i].Name = p.Name
			resBindings[i].Value, err = template.ParseJSONPath(de, p.Value)
			if err != nil {
				l.Warnf("get value of %s:%s failed %v. ", p.Name, p.Value, err)
				continue
			}
		}

		name := listener.EventListenerName
		namespace := listener.EventListenerNs
		if strings.Contains(name, "$(") {
			name = applyParamToResourceTemplate(resBindings, name)
		}
		if strings.Contains(namespace, "$(") {
			namespace = applyParamToResourceTemplate(resBindings, namespace)
		}

		key := GetListenerKey(name, namespace)

		elObj, ok := eventListenerMapInfo.Load(key)
		if !ok {
			l.Warnf("event listener %s not found, skip trigger. ", key)
			continue
		}
		el, ok := elObj.(*v1alpha1.EventListener)
		if !ok {
			l.Errorf("get unknown object  %s: %v. ", key, reflect.TypeOf(elObj))
			continue
		}
		if el.Status.Address == nil || el.Status.Address.URL == nil {
			l.Warnf("el %s is not ready, get empty url, skip. ", key)
			continue
		}
		reqUrl := el.Status.Address.URL.String()

		// resole headers
		resHeaders := make(map[string]string, len(listener.ReqHeadFields))
		for k, v := range listener.ReqHeadFields {
			if !strings.HasPrefix(v, "$(") {
				resHeaders[k] = v
				continue
			}
			resHeaders[k], err = template.ParseJSONPath(de, v)
			if err != nil {
				l.Warnf("get head value of %s:%s failed %v. ", k, v, err)
				continue
			}
		}

		// parse body template
		bodyMsg := applyParamToResourceTemplate(resBindings, listener.ReqBodyTemplate)

		ctx, span := trace.StartSpanWithRemoteParent(ctx, de.Context.GetSource(), parentSpanCtx)
		span.AddAttributes(trace.StringAttribute("eventID", de.Context.GetID()),
			trace.StringAttribute("source", de.Context.GetSource()),
		)
		span.Annotate([]trace.Attribute{
			trace.StringAttribute("context", de.Context.String()),
		}, "content")
		// do request
		reqCtx, _ := context.WithTimeout(ctx, time.Second*10)
		req, err := http.NewRequestWithContext(reqCtx, "POST", reqUrl, strings.NewReader(bodyMsg))
		if err != nil {
			l.Errorf("%s new request of %s failed %v. ", key, reqUrl, err)
			span.End()
			continue
		}
		for k, v := range resHeaders {
			req.Header.Set(k, v)
		}
		req.Header.Set("eventId", de.Context.GetID())
		req.Header.Set("traceBinaryId", base64.StdEncoding.EncodeToString(propagation.Binary(span.SpanContext())))

		if l.Desugar().Core().Enabled(zapcore.DebugLevel) {
			m, _ := httputil.DumpRequest(req, true)
			l.Debugf("%s do request %s. ", key, string(m))
		}

		resp, err := traceClient.Do(req)
		if err != nil {
			l.Errorf("%s do request %s failed %v. ", key, reqUrl, err)
			span.End()
			continue
		}
		msg, _ := ioutil.ReadAll(resp.Body)
		if resp.Status[0] == '2' {
			l.Infof("%s do request %s ok, response msg: %s. ", key, reqUrl, string(msg))
		} else {
			l.Errorf("%s do request %s failed, status:%s , msg: %s. ", key, reqUrl, resp.Status, string(msg))
		}
		_, rspSpan := trace.StartSpan(ctx, de.Context.GetSource()+"-resp")
		rspSpan.Annotate([]trace.Attribute{
			trace.StringAttribute("response", string(msg)),
		}, "response")
		resp.Body.Close()
		rspSpan.End()
		span.End()
	}

}

func applyParamToResourceTemplate(params []v1alpha1.Param, template string) string {
	res := template
	for _, param := range params {

		// Assume the param is valid
		paramVariable := fmt.Sprintf("$(params.%s)", param.Name)
		// Escape quotes so that that JSON strings can be appended to regular strings.
		// See #257 for discussion on this behavior.
		paramValue := strings.Replace(param.Value, `"`, `\"`, -1)
		res = strings.Replace(res, paramVariable, paramValue, -1)
	}
	return res
}
