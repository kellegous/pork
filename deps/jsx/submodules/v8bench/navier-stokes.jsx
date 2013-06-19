// -*- mode: jsx; jsx-indent-level: 4; indent-tabs-mode: nil; -*-
/**
 * Copyright 2012 the V8 project authors. All rights reserved.
 * Copyright 2009 Oliver Hunt <http://nerget.com>
 *
 * Permission is hereby granted, free of charge, to any person
 * obtaining a copy of this software and associated documentation
 * files (the "Software"), to deal in the Software without
 * restriction, including without limitation the rights to use,
 * copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the
 * Software is furnished to do so, subject to the following
 * conditions:
 *
 * The above copyright notice and this permission notice shall be
 * included in all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
 * EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
 * OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
 * NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
 * HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
 * WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
 * FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
 * OTHER DEALINGS IN THE SOFTWARE.
 */

import './base.jsx';

class NavierStokes {
    function constructor() {

        var solver = null : FluidField;

        function addPoints(field : Field) : void {
            var n = 64;
            for (var i = 1; i <= n; i++) {
                field.setVelocity(i, i, n, n);
                field.setDensity(i, i, 5);
                field.setVelocity(i, n - i, -n, -n);
                field.setDensity(i, n - i, 20);
                field.setVelocity(128 - i, n + i, -n, -n);
                field.setDensity(128 - i, n + i, 30);
            }
        }

        var framesTillAddingPoints = 0;
        var framesBetweenAddingPoints = 5;

        function prepareFrame(field : Field) : void
        {
            if (framesTillAddingPoints == 0) {
                addPoints(field);
                framesTillAddingPoints = framesBetweenAddingPoints;
                framesBetweenAddingPoints++;
            } else {
                framesTillAddingPoints--;
            }
        }

        function runNavierStokes() : void
        {
            solver.update();
        }

        function setupNavierStokes() : void
        {
            solver = new FluidField();
            solver.setResolution(128, 128);
            solver.setIterations(20);
            solver.setDisplayFunction(function(f : Field):void{});
            solver.setUICallback(prepareFrame);
            solver.reset();
        }

        function tearDownNavierStokes() : void
        {
            solver = null;
        }

        var navierStokes = new BenchmarkSuite('NavierStokes', 1484000, [
            new Benchmark('NavierStokes',
                runNavierStokes,
                setupNavierStokes,
                tearDownNavierStokes)
            ]);
    }
}


// Code from Oliver Hunt (http://nerget.com/fluidSim/pressure.js) starts here.
class FluidField {

    var _width : number;
    var _height : number;
    var _rowSize : number;

    var _iterations = 10;
    var _uiCallback = function(f : Field) : void {};
    var _displayFunc = null : (Field) -> void;

    var _visc = 0.5;
    var _dt = 0.1;
    var _dens = null : number[];
    var _dens_prev = null : number[];
    var _u = null : number[];
    var _u_prev = null : number[];
    var _v = null : number[];
    var _v_prev = null : number[];
    var _size = -1;

    function constructor() {
        this.setResolution(64, 64);
    }

    function _add_fields (x : number[], s : number[], dt : number) : void {
        for (var i=0; i<this._size ; i++ )
            x[i] += dt*s[i];
    }

    function _set_bnd(b : number, x : number[]) : void {
        if (b==1) {
            for (var i = 1; i <= this._width; i++) {
                x[i] =  x[i + this._rowSize];
                x[i + (this._height+1) *this._rowSize] = x[i + this._height * this._rowSize];
            }

            for (var j = 1; i <= this._height; i++) {
                x[j * this._rowSize] = -x[1 + j * this._rowSize];
                x[(this._width + 1) + j * this._rowSize] = -x[this._width + j * this._rowSize];
            }
        } else if (b == 2) {
            for (var i = 1; i <= this._width; i++) {
                x[i] = -x[i + this._rowSize];
                x[i + (this._height + 1) * this._rowSize] = -x[i + this._height * this._rowSize];
            }

            for (var j = 1; j <= this._height; j++) {
                x[j * this._rowSize] =  x[1 + j * this._rowSize];
                x[(this._width + 1) + j * this._rowSize] =  x[this._width + j * this._rowSize];
            }
        } else {
            for (var i = 1; i <= this._width; i++) {
                x[i] =  x[i + this._rowSize];
                x[i + (this._height + 1) * this._rowSize] = x[i + this._height * this._rowSize];
            }

            for (var j = 1; j <= this._height; j++) {
                x[j * this._rowSize] =  x[1 + j * this._rowSize];
                x[(this._width + 1) + j * this._rowSize] =  x[this._width + j * this._rowSize];
            }
        }
        var maxEdge = (this._height + 1) * this._rowSize;
        x[0]                       = 0.5 * (x[1] + x[this._rowSize]);
        x[maxEdge]                 = 0.5 * (x[1 + maxEdge] + x[this._height * this._rowSize]);
        x[(this._width+1)]         = 0.5 * (x[this._width] + x[(this._width + 1) + this._rowSize]);
        x[(this._width+1)+maxEdge] = 0.5 * (x[this._width + maxEdge] + x[(this._width + 1) + this._height * this._rowSize]);
    }

    function _lin_solve(b : number, x : number[], x0 : number[], a : number, c : number) : void {
        if (a == 0 && c == 1) {
            for (var j=1 ; j<=this._height; j++) {
                var currentRow = j * this._rowSize;
                ++currentRow;
                for (var i = 0; i < this._width; i++) {
                    x[currentRow] = x0[currentRow];
                    ++currentRow;
                }
            }
            this._set_bnd(b, x);
        } else {
            var invC = 1 / c;
            for (var k=0 ; k<this._iterations; k++) {
                for (var j=1 ; j<=this._height; j++) {
                    var lastRow = (j - 1) * this._rowSize;
                    var currentRow = j * this._rowSize;
                    var nextRow = (j + 1) * this._rowSize;
                    var lastX = x[currentRow];
                    ++currentRow;
                    for (var i=1; i<=this._width; i++)
                        lastX = x[currentRow] = (x0[currentRow] + a*(lastX+x[++currentRow]+x[++lastRow]+x[++nextRow])) * invC;
                }
                this._set_bnd(b, x);
            }
        }
    }

    function _diffuse(b : number, x : number[], x0 : number[], dt : number) : void {
        var a = 0;
        this._lin_solve(b, x, x0, a, 1 + 4*a);
    }

    function _lin_solve2(x : number[], x0 : number[], y : number[], y0 : number[], a : number, c : number) : void {
        if (a == 0 && c == 1) {
            for (var j=1 ; j <= this._height; j++) {
                var currentRow = j * this._rowSize;
                ++currentRow;
                for (var i = 0; i < this._width; i++) {
                    x[currentRow] = x0[currentRow];
                    y[currentRow] = y0[currentRow];
                    ++currentRow;
                }
            }
            this._set_bnd(1, x);
            this._set_bnd(2, y);
        } else {
            var invC = 1/c;
            for (var k=0 ; k<this._iterations; k++) {
                for (var j=1 ; j <= this._height; j++) {
                    var lastRow = (j - 1) * this._rowSize;
                    var currentRow = j * this._rowSize;
                    var nextRow = (j + 1) * this._rowSize;
                    var lastX = x[currentRow];
                    var lastY = y[currentRow];
                    ++currentRow;
                    for (var i = 1; i <= this._width; i++) {
                        lastX = x[currentRow] = (x0[currentRow] + a * (lastX + x[currentRow] + x[lastRow] + x[nextRow])) * invC;
                        lastY = y[currentRow] = (y0[currentRow] + a * (lastY + y[++currentRow] + y[++lastRow] + y[++nextRow])) * invC;
                    }
                }
                this._set_bnd(1, x);
                this._set_bnd(2, y);
            }
        }
    }

    function _diffuse2(x : number[], x0 : number[], y : number[], y0 : number[], dt : number) : void {
        var a = 0;
        this._lin_solve2(x, x0, y, y0, a, 1 + 4 * a);
    }

    function _advect(b : number, d : number[], d0 : number[], u : number[], v : number[], dt : number) : void {
        var Wdt0 = dt * this._width;
        var Hdt0 = dt * this._height;
        var Wp5 = this._width + 0.5;
        var Hp5 = this._height + 0.5;
        for (var j = 1; j<= this._height; j++) {
            var pos = j * this._rowSize;
            for (var i = 1; i <= this._width; i++) {
                var x = i - Wdt0 * u[++pos];
                var y = j - Hdt0 * v[pos];
                if (x < 0.5)
                    x = 0.5;
                else if (x > Wp5)
                    x = Wp5;
                var i0 = x | 0;
                var i1 = i0 + 1;
                if (y < 0.5)
                    y = 0.5;
                else if (y > Hp5)
                    y = Hp5;
                var j0 = y | 0;
                var j1 = j0 + 1;
                var s1 = x - i0;
                var s0 = 1 - s1;
                var t1 = y - j0;
                var t0 = 1 - t1;
                var row1 = j0 * this._rowSize;
                var row2 = j1 * this._rowSize;
                d[pos] = s0 * (t0 * d0[i0 + row1] + t1 * d0[i0 + row2]) + s1 * (t0 * d0[i1 + row1] + t1 * d0[i1 + row2]);
            }
        }
        this._set_bnd(b, d);
    }

    function _project(u : number[], v : number[], p : number[], div : number[]) : void {
        var h = -0.5 / Math.sqrt(this._width * this._height);
        for (var j = 1 ; j <= this._height; j++ ) {
            var row = j * this._rowSize;
            var previousRow = (j - 1) * this._rowSize;
            var prevValue = row - 1;
            var currentRow = row;
            var nextValue = row + 1;
            var nextRow = (j + 1) * this._rowSize;
            for (var i = 1; i <= this._width; i++ ) {
                div[++currentRow] = h * (u[++nextValue] - u[++prevValue] + v[++nextRow] - v[++previousRow]);
                p[currentRow] = 0;
            }
        }
        this._set_bnd(0, div);
        this._set_bnd(0, p);

        this._lin_solve(0, p, div, 1, 4 );
        var wScale = 0.5 * this._width;
        var hScale = 0.5 * this._height;
        for (var j = 1; j<= this._height; j++ ) {
            var prevPos = j * this._rowSize - 1;
            var currentPos = j * this._rowSize;
            var nextPos = j * this._rowSize + 1;
            var prevRow = (j - 1) * this._rowSize;
            var currentRow = j * this._rowSize;
            var nextRow = (j + 1) * this._rowSize;

            for (var i = 1; i<= this._width; i++) {
                u[++currentPos] -= wScale * (p[++nextPos] - p[++prevPos]);
                v[currentPos]   -= hScale * (p[++nextRow] - p[++prevRow]);
            }
        }
        this._set_bnd(1, u);
        this._set_bnd(2, v);
    }

    function _dens_step(x : number[], x0 : number[], u : number[], v : number[], dt : number) : void {
        this._add_fields(x, x0, dt);
        this._diffuse(0, x0, x, dt );
        this._advect(0, x, x0, u, v, dt );
    }

    function _vel_step(u : number[], v : number[], u0 : number[], v0 : number[], dt : number) : void {
        this._add_fields(u, u0, dt );
        this._add_fields(v, v0, dt );
        var temp = u0; u0 = u; u = temp;
        var temp = v0; v0 = v; v = temp;
        this._diffuse2(u,u0,v,v0, dt);
        this._project(u, v, u0, v0);
        var temp = u0; u0 = u; u = temp;
        var temp = v0; v0 = v; v = temp;
        this._advect(1, u, u0, u0, v0, dt);
        this._advect(2, v, v0, u0, v0, dt);
        this._project(u, v, u0, v0 );
    }

    function update () : void {
        this.queryUI(this._dens_prev, this._u_prev, this._v_prev);
        this._vel_step(this._u, this._v, this._u_prev, this._v_prev, this._dt);
        this._dens_step(this._dens, this._dens_prev, this._u, this._v, this._dt);
        this._displayFunc(new Field(this, this._dens, this._u, this._v));
    }

    function iterations () : number {
        return this._iterations;
    }

    function setIterations (iters : number) : void {
        if (iters > 0 && iters <= 100)
            this._iterations = iters;
    }

    function queryUI (d : number[], u : number[], v : number[]) : void {
        for (var i = 0; i < this._size; i++)
            u[i] = v[i] = d[i] = 0.0;
        this._uiCallback(new Field(this, d, u, v));
    }

    function setUICallback (callback : (Field) -> void) : void {
        this._uiCallback = callback;
    }

    function setDisplayFunction (func : (Field) -> void) : void {
        this._displayFunc = func;
    }
    
    function reset () : void {
        this._rowSize = this._width + 2;
        this._size = (this._width+2)*(this._height+2);
        this._dens = new Array.<number>(this._size);
        this._dens_prev = new Array.<number>(this._size);
        this._u = new Array.<number>(this._size);
        this._u_prev = new Array.<number>(this._size);
        this._v = new Array.<number>(this._size);
        this._v_prev = new Array.<number>(this._size);
        for (var i = 0; i < this._size; i++)
            this._dens_prev[i] = this._u_prev[i] = this._v_prev[i] = this._dens[i] = this._u[i] = this._v[i] = 0;
    }

    function setResolution (hRes : number, wRes : number) : boolean {
        var res = wRes * hRes;
        if (res > 0 && res < 1000000 && (wRes != this._width || hRes != this._height)) {
            this._width = wRes;
            this._height = hRes;
            this.reset();
            return true;
        }
        return false;
    }

}

class Field {

    var fluid : FluidField;
    var dens : number[];
    var u : number[];
    var v : number[];

    function constructor(fluid : FluidField, dens : number[], u : number[], v : number[]) {
        this.fluid = fluid;
        this.dens = dens;
        this.u = u;
        this.v = v;
    }

    function setDensity(x : number, y : number, d : number) : void {
        this.dens[(x + 1) + (y + 1) * this.fluid._rowSize] = d;
    }

    function getDensity(x : number, y : number) : number {
        return this.dens[(x + 1) + (y + 1) * this.fluid._rowSize];
    }

    function setVelocity(x : number, y : number, xv : number, yv : number) : void {
        this.u[(x + 1) + (y + 1) * this.fluid._rowSize] = xv;
        this.v[(x + 1) + (y + 1) * this.fluid._rowSize] = yv;
    }

    function getXVelocity(x : number, y : number) : number {
        return this.u[(x + 1) + (y + 1) * this.fluid._rowSize];
    }

    function getYVelocity(x : number, y : number) : number {
        return this.v[(x + 1) + (y + 1) * this.fluid._rowSize];
    }

    function width() : number {
	    return this.fluid._width;
    }

    function height() : number {
	    return this.fluid._height;
    }

}

// vim: set expandtab:
