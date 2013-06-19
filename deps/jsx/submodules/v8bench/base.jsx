// -*- mode: jsx; jsx-indent-level: 4; indent-tabs-mode: nil; -*-
// Copyright 2012 the V8 project authors. All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
//       notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
//       copyright notice, this list of conditions and the following
//       disclaimer in the documentation and/or other materials provided
//       with the distribution.
//     * Neither the name of Google Inc. nor the names of its
//       contributors may be used to endorse or promote products derived
//       from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.


import "timer.jsx";


// Simple framework for running the benchmark suites and
// computing a score based on the timing measurements.

class Benchmark {

    var name     : string;
    var run      : () -> void;
    var Setup    : () -> void;
    var TearDown : () -> void;

    // A benchmark has a name (string) and a function that will be run to
    // do the performance measurement. The optional setup and tearDown
    // arguments are functions that will be invoked before and after
    // running the benchmark, but the running time of these functions will
    // not be accounted for in the benchmark score.
    function constructor(name : string, run : () -> void) {
        this(name, run, function() { }, function() { });
    }

    function constructor(name : string, run : () -> void, setup : () -> void, tearDown : () -> void) {
        this.name = name;
        this.run = run;
        this.Setup = setup;
        this.TearDown = tearDown;
    }
}

class BenchmarkResult {

    var benchmark        : Benchmark;
    var time                : number;

    // Benchmark results hold the benchmark and the measured time used to
    // run the benchmark. The benchmark score is computed later once a
    // full benchmark suite has run to completion.
    function constructor(benchmark : Benchmark, time : number) {
        this.benchmark = benchmark;
        this.time = time;
    }

    // Automatically convert results to numbers. Used by the geometric
    // mean computation.
    function valueOf() : number {
        return this.time;
    }
}

class BenchmarkData {

    var runs : number;                // FIXME: wasabiz
    var elapsed : number;

    function constructor(runs : number, elapsed : number) {
        this.runs = runs;
        this.elapsed = elapsed;
    }
}

class BenchmarkRunner {
    function NotifyStart(name : string) : void {}
    function NotifyStep(name : string) : void {}
    function NotifyResult(name : string, result : string) : void {}
    function NotifyError(name : string, error : Error) : void {}
    function NotifyScore(score : string) : void {}
}

class BenchmarkUtil {

    static var seed = 49734321;

    // To make the benchmark results predictable, we replace Math.random
    // with a 100% deterministic alternative.
    static function random() : number {
        // Robert Jenkins' 32 bit integer hash function.
        BenchmarkUtil.seed = ((BenchmarkUtil.seed + 0x7ed55d16) + (BenchmarkUtil.seed << 12))  & 0xffffffff;
        BenchmarkUtil.seed = ((BenchmarkUtil.seed ^ 0xc761c23c) ^ (BenchmarkUtil.seed >>> 19)) & 0xffffffff;
        BenchmarkUtil.seed = ((BenchmarkUtil.seed + 0x165667b1) + (BenchmarkUtil.seed << 5))   & 0xffffffff;
        BenchmarkUtil.seed = ((BenchmarkUtil.seed + 0xd3a2646c) ^ (BenchmarkUtil.seed << 9))   & 0xffffffff;
        BenchmarkUtil.seed = ((BenchmarkUtil.seed + 0xfd7046c5) + (BenchmarkUtil.seed << 3))   & 0xffffffff;
        BenchmarkUtil.seed = ((BenchmarkUtil.seed ^ 0xb55a4f09) ^ (BenchmarkUtil.seed >>> 16)) & 0xffffffff;
        return (BenchmarkUtil.seed & 0xfffffff) / 0x10000000;
    }
}

class BenchmarkSuite {

    // Keep track of all declared benchmark suites.
    static var suites = [] : BenchmarkSuite[];


    static var scores = [] : number[];


    var results = [] : BenchmarkResult[];
    var runner : BenchmarkRunner;


    // Scores are not comparable across versions. Bump the version if
    // you're making changes that will affect that scores, e.g. if you add
    // a new benchmark or change an existing one.
    static var version = '7';


    var name       : string;
    var reference  : int;
    var benchmarks : Benchmark[];


    // Suites of benchmarks consist of a name and the set of benchmarks in
    // addition to the reference timing that the final score will be based
    // on. This way, all scores are relative to a reference run and higher
    // scores implies better performance.
    function constructor(name : string, reference : int, benchmarks : Benchmark[]) {
        this.name = name;
        this.reference = reference;
        this.benchmarks = benchmarks;
        BenchmarkSuite.suites.push(this);
    }

    // Runs all registered benchmark suites and optionally yields between
    // each individual benchmark to avoid running for too long in the
    // context of browsers. Once done, the final score is reported to the
    // runner.
    static function RunSuites(runner : BenchmarkRunner) : void {
        var continuation = null : () -> variant;
        var suites = BenchmarkSuite.suites;
        var length = suites.length;
        BenchmarkSuite.scores = [] : number[];
        var index = 0;
        function RunStep() : void {
            while (continuation || index < length) {
                if (continuation) {
                    continuation = continuation() as () -> variant;
                } else {
                    var suite = suites[index++];
                    runner.NotifyStart(suite.name);
                    continuation = suite.RunStep(runner) as () -> variant;
                }
                if (continuation) {
                    Timer.setTimeout(RunStep, 25);
                    return;
                }
            }
            var score = BenchmarkSuite.GeometricMean(BenchmarkSuite.scores);
            var formatted = BenchmarkSuite.FormatScore(100 * score);
            runner.NotifyScore(formatted);
        }
        RunStep();
    }


    // Counts the total number of registered benchmarks. Useful for
    // showing progress as a percentage.
    static function CountBenchmarks() : int {
        var result = 0;
        var suites = BenchmarkSuite.suites;
        for (var i = 0; i < suites.length; i++) {
            result += suites[i].benchmarks.length;
        }
        return result;
    }


    // Computes the geometric mean of a set of numbers.
    static function GeometricMean(numbers : number[]) : number {
        var loga = 0;
        for (var i = 0; i < numbers.length; i++) {
            loga += Math.log(numbers[i].valueOf());
        }
        return Math.pow(Math.E, loga / numbers.length);
    }

    static function GeometricMean(numbers : BenchmarkResult[]) : number {
        var loga = 0;
        for (var i = 0; i < numbers.length; i++) {
            loga += Math.log(numbers[i].valueOf());
        }
        return Math.pow(Math.E, loga / numbers.length);
    }


    // Converts a score value to a string with at least three significant
    // digits.
    static function FormatScore(value : number) : string {
        if (value > 100) {
            return value.toFixed(0);
        } else {
            return value.toPrecision(3);
        }
    }

    // Notifies the runner that we're done running a single benchmark in
    // the benchmark suite. This can be useful to report progress.
    function NotifyStep(result : BenchmarkResult) : void {
        this.results.push(result);
        this.runner.NotifyStep(result.benchmark.name);
    }


    // Notifies the runner that we're done with running a suite and that
    // we have a result which can be reported to the user if needed.
    function NotifyResult() : void {
        var mean = BenchmarkSuite.GeometricMean(this.results);
        var score = this.reference / mean;
        BenchmarkSuite.scores.push(score);

        var formatted = BenchmarkSuite.FormatScore(100 * score);
        this.runner.NotifyResult(this.name, formatted);
    }


    // Notifies the runner that running a benchmark resulted in an error.
    function NotifyError(error : Error) : void {
        this.runner.NotifyError(this.name, error);
        this.runner.NotifyStep(this.name);
    }


    // Runs a single benchmark for at least a second and computes the
    // average time it takes to run a single iteration.
    function RunSingleBenchmark(benchmark : Benchmark, data : BenchmarkData) : BenchmarkData {
        function Measure(data : BenchmarkData) : void {
            var elapsed = 0;
            var start = new Date();
            for (var n = 0; elapsed < 1000; n++) {
                benchmark.run();
                elapsed = new Date().valueOf() - start.valueOf();
            }
            if (data != null) {
                data.runs += n;
                data.elapsed += elapsed;
            }
        }

        if (data == null) {
            // Measure the benchmark once for warm up and throw the result
            // away. Return a fresh data object.
            Measure(null);
            return new BenchmarkData(0, 0);
        } else {
            Measure(data);
            // If we've run too few iterations, we continue for another second.
            if (data.runs < 32) return data;
            var usec = (data.elapsed * 1000) / data.runs;
            this.NotifyStep(new BenchmarkResult(benchmark, usec));
            return null;
        }
    }


    // This function starts running a suite, but stops between each
    // individual benchmark in the suite and returns a continuation
    // function which can be invoked to run the next benchmark. Once the
    // last benchmark has been executed, null is returned.
    function RunStep(runner : BenchmarkRunner) : variant {
        this.results = [] : BenchmarkResult[];
        this.runner = runner;
        var length = this.benchmarks.length;
        var index = 0;
        var suite = this;
        var data = null : BenchmarkData;

        // Run the setup, the actual benchmark, and the tear down in three
        // separate steps to allow the framework to yield between any of the
        // steps.

        var RunNextSetup        = null : () -> variant;
        var RunNextBenchmark        = null : () -> variant;
        var RunNextTearDown        = null : () -> variant;

        RunNextSetup =         function () : variant {
            if (index < length) {
                try {
                    suite.benchmarks[index].Setup();
                } catch (e : Error) {
                    suite.NotifyError(e);
                    return null;
                }
                return RunNextBenchmark;
            }
            suite.NotifyResult();
            return null;
        };

        RunNextBenchmark = function () : variant {
            try {
                data = suite.RunSingleBenchmark(suite.benchmarks[index], data);
            } catch (e : Error) {
                suite.NotifyError(e);
                return null;
            }
            // If data is null, we're done with this benchmark.
            return (data == null) ? RunNextTearDown as variant : RunNextBenchmark();
        };

        RunNextTearDown = function () : variant {
            try {
                suite.benchmarks[index++].TearDown();
            } catch (e : Error) {
                suite.NotifyError(e);
                return null;
            }
            return RunNextSetup;
        };

        // Start out running the setup.
        return RunNextSetup();
    }
}
// vim: set expandtab:
