package main

import (
	"os"

	"github.com/filecoin-project/go-filecoin/commands"

	logging "gx/ipfs/QmRb5jh8z2E8hMGN2tkvs1yHynUanqnZ3UeKwgN1i9P1F8/go-log"

	/* USED FOR JAEGER TRACER */
	//opentracing "github.com/opentracing/opentracing-go"
	//"github.com/uber/jaeger-client-go/config"

	/* USED FOR GO-LOG TRACER */
	opentracing "gx/ipfs/QmWLWmRVSiagqP15jczsGME1qpob6HDbtbHAY2he9W5iUo/opentracing-go"
	// TODO This gx package isn't published yet
	tracer "gx/ipfs/Qmaf59ke1Gu4rz9tP8MzCp6PyGv9ZU9cNJvPwrwNavSL9r/go-log/tracer"
)

func main() {

	// TODO make this a plugin, and configration option
	/* Jaeger Tracer */
	/*
		tracerCfg := &config.Configuration{
			Sampler: &config.SamplerConfig{
				Type:  "const",
				Param: 1,
			},
			Reporter: &config.ReporterConfig{
				LogSpans: true,
			},
		}
		//we are ignoring the closer for now
		tracer, _, err := tracerCfg.New("go-filecoin")
		if err != nil {
			panic(err)
		}
	*/

	/* go-log Tracer */
	lgblRecorder := tracer.NewLoggableRecorder()
	tracer := tracer.New(lgblRecorder)

	opentracing.SetGlobalTracer(tracer)

	// TODO: make configurable
	// TODO: find a better home for this
	logging.Configure(logging.LevelDebug)

	// TODO implement help text like so:
	// https://github.com/ipfs/go-ipfs/blob/master/core/commands/root.go#L91
	// TODO don't panic if run without a command.
	code, _ := commands.Run(os.Args, os.Stdin, os.Stdout, os.Stderr)
	os.Exit(code)
}
