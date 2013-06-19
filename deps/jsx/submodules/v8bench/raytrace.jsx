// -*- mode: jsx; jsx-indent-level: 4; indent-tabs-mode: nil; -*-
// The ray tracer code in this file is written by Adam Burmister. It
// is available in its original form from:
//
//   http://labs.flog.nz.co/raytracer/
//
// It has been modified slightly by Google to work as a standalone
// benchmark, but the all the computational code remains
// untouched. This file also contains a copy of parts of the Prototype
// JavaScript framework which is used by the ray tracer.

import "./base.jsx";
import "js/web.jsx";

class RayTrace {
    function constructor() {
        var rayTrace = new BenchmarkSuite('RayTrace', 739989, [
            new Benchmark('RayTrace', function() { Main.renderScene(); })
            ]);
    }
}


// ------------------------------------------------------------------------
// ------------------------------------------------------------------------

// The rest of this file is the actual ray tracer written by Adam
// Burmister. It's a concatenation of the following files:
//
//   flog/color.js
//   flog/light.js
//   flog/vector.js
//   flog/ray.js
//   flog/scene.js
//   flog/material/basematerial.js
//   flog/material/solid.js
//   flog/material/chessboard.js
//   flog/shape/baseshape.js
//   flog/shape/sphere.js
//   flog/shape/plane.js
//   flog/intersectioninfo.js
//   flog/camera.js
//   flog/background.js
//   flog/engine.js


// This class used to be in Flog.RayTracer namespace
class Color {

    var red     = 0.0;
    var green   = 0.0;
    var blue    = 0.0;

    function constructor() {
        this(0.0, 0.0, 0.0);
    }

    function constructor(r : number, g : number, b : number) {
        this.red = r;
        this.green = g;
        this.blue = b;
    }

    static function add(c1 : Color, c2 : Color) : Color {
        var result = new Color(0,0,0);

        result.red = c1.red + c2.red;
        result.green = c1.green + c2.green;
        result.blue = c1.blue + c2.blue;

        return result;
    }

    static function addScalar(c1 : Color, s : number) : Color {
        var result = new Color(0,0,0);

        result.red = c1.red + s;
        result.green = c1.green + s;
        result.blue = c1.blue + s;

        result.limit();

        return result;
    }

    static function subtract(c1 : Color, c2 : Color) : Color {
        var result = new Color(0,0,0);

        result.red = c1.red - c2.red;
        result.green = c1.green - c2.green;
        result.blue = c1.blue - c2.blue;

        return result;
    }

    static function multiply(c1 : Color, c2 : Color) : Color {
        var result = new Color(0,0,0);

        result.red = c1.red * c2.red;
        result.green = c1.green * c2.green;
        result.blue = c1.blue * c2.blue;

        return result;
    }

    static function multiplyScalar(c1 : Color, f : number) : Color {
        var result = new Color(0,0,0);

        result.red = c1.red * f;
        result.green = c1.green * f;
        result.blue = c1.blue * f;

        return result;
    }

    static function divideFactor(c1 : Color, f : number) : Color {
        var result = new Color(0,0,0);

        result.red = c1.red / f;
        result.green = c1.green / f;
        result.blue = c1.blue / f;

        return result;
    }

    function limit() : void {
        this.red = (this.red > 0.0) ? ( (this.red > 1.0) ? 1.0 : this.red ) : 0.0;
        this.green = (this.green > 0.0) ? ( (this.green > 1.0) ? 1.0 : this.green ) : 0.0;
        this.blue = (this.blue > 0.0) ? ( (this.blue > 1.0) ? 1.0 : this.blue ) : 0.0;
    }

    function distance(color : Color) : number {
        var d = Math.abs(this.red - color.red) + Math.abs(this.green - color.green) + Math.abs(this.blue - color.blue);
        return d;
    }

    static function blend(c1 : Color, c2 : Color, w : number) : Color {
        var result = new Color(0,0,0);
        result = Color.add(
            Color.multiplyScalar(c1, 1 - w),
            Color.multiplyScalar(c2, w)
        );
        return result;
    }

    function brightness() : number {
        var r = Math.floor(this.red*255);
        var g = Math.floor(this.green*255);
        var b = Math.floor(this.blue*255);
        return (r * 77 + g * 150 + b * 29) >> 8;
    }

    override function toString() : string {
        var r = Math.floor(this.red*255);
        var g = Math.floor(this.green*255);
        var b = Math.floor(this.blue*255);

        return "rgb("+ r as string +","+ g as string +","+ b as string +")";
    }
}

class Light {

    var position        = null : Vector;
    var color           = null : Color;
    var intensity       = 10.0;

    function constructor(pos : Vector, color : Color) {
        this(pos, color, 10.0);
    }

    function constructor(pos : Vector, color : Color, intensity : number) {
        this.position = pos;
        this.color = color;
        this.intensity = intensity;
    }

    override function toString() : string {
        return 'Light [' + this.position.x as string + ',' + this.position.y as string + ',' + this.position.z as string + ']';
    }
}

class Vector {

    var x = 0.0;
    var y = 0.0;
    var z = 0.0;

    function constructor() {
        this(0,0,0);
    }

    function constructor(x : number, y : number, z : number) {
        this.x = x;
        this.y = y;
        this.z = z;
    }

    function copy(vector : Vector) : void {
        this.x = vector.x;
        this.y = vector.y;
        this.z = vector.z;
    }

    function normalize() : Vector {
        var m = this.magnitude();
        return new Vector(this.x / m, this.y / m, this.z / m);
    }

    function magnitude() : number {
        return Math.sqrt((this.x * this.x) + (this.y * this.y) + (this.z * this.z));
    }

    function cross(w : Vector) : Vector{
        return new Vector(
            -this.z * w.y + this.y * w.z,
            this.z * w.x - this.x * w.z,
            -this.y * w.x + this.x * w.y);
    }

    function dot(w : Vector) : number {
        return this.x * w.x + this.y * w.y + this.z * w.z;
    }

    static function add(v : Vector, w : Vector) : Vector {
        return new Vector(w.x + v.x, w.y + v.y, w.z + v.z);
    }

    static function subtract(v : Vector, w : Vector) : Vector {
        if(!w || !v) throw 'Vectors must be defined [' + v.toString() + ',' + w.toString() + ']';
        return new Vector(v.x - w.x, v.y - w.y, v.z - w.z);
    }

    static function multiplyVector(v : Vector, w : Vector) : Vector {
        return new Vector(v.x * w.x, v.y * w.y, v.z * w.z);
    }

    static function multiplyScalar(v : Vector, w : number) : Vector {
        return new Vector(v.x * w, v.y * w, v.z * w);
    }

    override function toString() : string {
        return 'Vector [' + this.x as string + ',' + this.y as string + ',' + this.z as string + ']';
    }
}

class Ray {

    var position        = null : Vector;
    var direction       = null : Vector;

    function constructor(pos : Vector, dir : Vector) {
        this.position = pos;
        this.direction = dir;
    }

    override function toString() : string {
        return 'Ray [' + this.position.toString() + ',' + this.direction.toString() + ']';
    }
}

/* Fake a Flog.* namespace */
class Scene {

    var camera          : Camera;
    var shapes          : Shape[];
    var lights          : Light[];
    var background      : Background;

    function constructor() {
        this.camera = new Camera(
            new Vector(0,0,-5),
            new Vector(0,0,1),
            new Vector(0,1,0)
        );
        this.shapes = new Array.<Shape>();
        this.lights = new Array.<Light>();
        this.background = new Background(new Color(0,0,0.5), 0.2);
    }
}

// NOTE
// BaseMaterial, Solid, and Chessboard classes used to be in Material namespace

abstract class BaseMaterial {

    var gloss           = 2.0;  // [0...infinity] 0 = matt
    var transparency    = 0.0;  // 0=opaque
    var reflection      = 0.0;  // [0...infinity] 0 = no reflection
    var refraction      = 0.50;
    var hasTexture      = false;

    function constructor() {

    }

    abstract function getColor(u : number, v : number) : Color;

    function wrapUp(t : number) : number {
        t = t % 2.0;
        if(t < -1) t += 2.0;
        if(t >= 1) t -= 2.0;
        return t;
    }

    override function toString() : string {
        return 'Material [gloss=' + this.gloss as string + ', transparency=' + this.transparency as string + ', hasTexture=' + this.hasTexture as string +']';
    }
}

class Solid extends BaseMaterial {

    var color : Color;

    function constructor(color : Color, reflection : number, refraction : number, transparency : number, gloss : number) {
        this.color = color;
        this.reflection = reflection;
        this.transparency = transparency;
        this.gloss = gloss;
        this.hasTexture = false;
    }

    override function getColor(u : number, v : number) : Color {
        return this.color;
    }

    override function toString() : string {
        return 'SolidMaterial [gloss=' + this.gloss as string + ', transparency=' + this.transparency as string + ', hasTexture=' + this.hasTexture as string +']';
    }
}

/* Fake a Flog.* namespace */
class Chessboard extends BaseMaterial {

    var colorEven       = null : Color;
    var colorOdd        = null : Color;
    var density         = 0.5;

    function constructor(colorEven : Color, colorOdd : Color, reflection : number, transparency : number, gloss : number, density : number) {
        this.colorEven = colorEven;
        this.colorOdd = colorOdd;
        this.reflection = reflection;
        this.transparency = transparency;
        this.gloss = gloss;
        this.density = density;
        this.hasTexture = true;
    }

    override function getColor(u : number, v : number) : Color {
        var t = this.wrapUp(u * this.density) * this.wrapUp(v * this.density);

        if(t < 0.0)
            return this.colorEven;
        else
            return this.colorOdd;
    }

    override function toString() : string {
        return 'ChessMaterial [gloss=' + this.gloss as string + ', transparency=' + this.transparency as string + ', hasTexture=' + this.hasTexture as string +']';
    }
}

// NOTE
// Sphere and Plane classes used to be in Shape namespace

abstract class Shape {

    var position : Vector;
    var material : BaseMaterial;

    abstract function intersect(ray : Ray) : IntersectionInfo;
}

class Sphere extends Shape {

    var radius : number;

    function constructor(pos : Vector, radius : number, material : BaseMaterial) {
        this.radius = radius;
        this.position = pos;
        this.material = material;
    }

    override function intersect(ray : Ray) : IntersectionInfo {
        var info = new IntersectionInfo();
        info.shape = this;

        var dst = Vector.subtract(ray.position, this.position);

        var B = dst.dot(ray.direction);
        var C = dst.dot(dst) - (this.radius * this.radius);
        var D = (B * B) - C;

        if(D > 0){ // intersection!
            info.isHit = true;
            info.distance = (-B) - Math.sqrt(D);
            info.position = Vector.add(
                ray.position,
                Vector.multiplyScalar(
                    ray.direction,
                    info.distance
                )
            );
            info.normal = Vector.subtract(
                info.position,
                this.position
            ).normalize();

            info.color = this.material.getColor(0,0);
        } else {
            info.isHit = false;
        }
        return info;
    }

    override function toString() : string {
        return 'Sphere [position=' + this.position.toString() + ', radius=' + this.radius as string + ']';
    }
}

class Plane extends Shape {

    var d               = 0.0;

    function constructor(pos : Vector, d : number, material : BaseMaterial) {
        this.position = pos;
        this.d = d;
        this.material = material;
    }

    override function intersect(ray : Ray) : IntersectionInfo {
        var info = new IntersectionInfo();

        var Vd = this.position.dot(ray.direction);
        if(Vd == 0) return info; // no intersection

        var t = -(this.position.dot(ray.position) + this.d) / Vd;
        if(t <= 0) return info;

        info.shape = this;
        info.isHit = true;
        info.position = Vector.add(
            ray.position,
            Vector.multiplyScalar(
                ray.direction,
                t
            )
        );
        info.normal = this.position;
        info.distance = t;

        if(this.material.hasTexture){
            var vU = new Vector(this.position.y, this.position.z, -this.position.x);
            var vV = vU.cross(this.position);
            var u = info.position.dot(vU);
            var v = info.position.dot(vV);
            info.color = this.material.getColor(u,v);
        } else {
            info.color = this.material.getColor(0,0);
        }

        return info;
    }

    override function toString() : string {
        return 'Plane [' + this.position.toString() + ', d=' + this.d as string + ']';
    }
}

class IntersectionInfo {

    var isHit           = false;
    var hitCount        = 0;
    var shape           = null : Shape;
    var position        = null : Vector;
    var normal          = null : Vector;
    var color           = null : Color;
    var distance        = null : Nullable.<number>;

    function constructor() {
        this.color = new Color(0,0,0);
    }

    override function toString() : string {
        return 'Intersection [' + this.position.toString() + ']';
    }
}


class Camera {

    var position        = null : Vector;
    var lookAt          = null : Vector;
    var equator         = null : Vector;
    var up              = null : Vector;
    var screen          = null : Vector;

    function constructor(pos : Vector, lookAt : Vector, up : Vector) {
        this.position = pos;
        this.lookAt = lookAt;
        this.up = up;
        this.equator = lookAt.normalize().cross(this.up);
        this.screen = Vector.add(this.position, this.lookAt);
    }

    function getRay(vx : number, vy : number) : Ray {
        var pos = Vector.subtract(
            this.screen,
            Vector.subtract(
                Vector.multiplyScalar(this.equator, vx),
                Vector.multiplyScalar(this.up, vy)
            )
        );
        pos.y = pos.y * -1;
        var dir = Vector.subtract(
            pos,
            this.position
        );

        var ray = new Ray(pos, dir.normalize());

        return ray;
    }

    override function toString() : string {
        return 'Ray []';
    }
}

/* Fake a Flog.* namespace */
class Background {

    var color = null : Color;
    var ambience = 0.0;

    function constructor(color : Color, ambience : number) {
        this.color = color;
        this.ambience = ambience;
    }
}

class Options {
    var canvasHeight            = 100;
    var canvasWidth             = 100;
    var pixelWidth              = 2;
    var pixelHeight             = 2;
    var renderDiffuse           = false;
    var renderShadows           = false;
    var renderHighlights        = false;
    var renderReflections       = false;
    var rayDepth                = 2;
 }

class Engine {

    var canvas = null : CanvasRenderingContext2D; /* 2d context we can render to */
    var options = null : Options;

    function constructor(options : Options) {
        this.options = options;
        this.options.canvasHeight /= this.options.pixelHeight;
        this.options.canvasWidth /= this.options.pixelWidth;

        /* TODO: dynamically include other scripts */
    }

    function setPixel(x : number, y : number, color : Color) : void {
        var pxW = this.options.pixelWidth;
        var pxH = this.options.pixelHeight;

        if (this.canvas) {
            this.canvas.fillStyle = color.toString();
            this.canvas.fillRect (x * pxW, y * pxH, pxW, pxH);
        } else {
            if (x == y) {
                Main.checkNumber += color.brightness();
            }
            // print(x * pxW, y * pxH, pxW, pxH);
        }
    }

    function renderScene(scene : Scene, canvas : HTMLCanvasElement) : void {
        Main.checkNumber = 0;
        /* Get canvas */
        if (canvas) {
            this.canvas = canvas.getContext("2d") as CanvasRenderingContext2D;
        } else {
            this.canvas = null;
        }

        var canvasHeight = this.options.canvasHeight;
        var canvasWidth = this.options.canvasWidth;

        for(var y=0; y < canvasHeight; y++){
            for(var x=0; x < canvasWidth; x++){
                var yp = y * 1.0 / canvasHeight * 2 - 1;
                var xp = x * 1.0 / canvasWidth * 2 - 1;

                var ray = scene.camera.getRay(xp, yp);

                var color = this.getPixelColor(ray, scene);

                this.setPixel(x, y, color);
            }
        }
        if (Main.checkNumber != 2321) {
            throw new Error("Scene rendered incorrectly");
        }
    }

    function getPixelColor(ray : Ray, scene : Scene) : Color {
        var info = this.testIntersection(ray, scene, null);
        if(info.isHit){
            var color = this.rayTrace(info, ray, scene, 0);
            return color;
        }
        return scene.background.color;
    }

    function testIntersection(ray : Ray, scene : Scene, exclude : Shape) : IntersectionInfo {
        var hits = 0;
        var best = new IntersectionInfo();
        best.distance = 2000;

        for(var i=0; i<scene.shapes.length; i++){
            var shape = scene.shapes[i];

            if(shape != exclude){
                var info = shape.intersect(ray);
                if(info.isHit && info.distance >= 0 && info.distance < best.distance){
                    best = info;
                    hits++;
                }
            }
        }
        best.hitCount = hits;
        return best;
    }

    function getReflectionRay(P : Vector ,N : Vector,V : Vector) : Ray {
        var c1 = -N.dot(V);
        var R1 = Vector.add(
            Vector.multiplyScalar(N, 2*c1),
            V
        );
        return new Ray(P, R1);
    }

    function rayTrace(info : IntersectionInfo, ray : Ray, scene : Scene, depth : number) : Color {
        // Calc ambient
        var color = Color.multiplyScalar(info.color, scene.background.ambience);
        var oldColor = color;
        var shininess = Math.pow(10, info.shape.material.gloss + 1);

        for(var i=0; i<scene.lights.length; i++){
            var light = scene.lights[i];

            // Calc diffuse lighting
            var v = Vector.subtract(
                light.position,
                info.position
            ).normalize();

            if(this.options.renderDiffuse){
                var L = v.dot(info.normal);
                if(L > 0.0){
                    color = Color.add(
                        color,
                        Color.multiply(
                            info.color,
                            Color.multiplyScalar(
                                light.color,
                                L
                            )
                        )
                    );
                }
            }

            // The greater the depth the more accurate the colours, but
            // this is exponentially (!) expensive
            if(depth <= this.options.rayDepth){
                // calculate reflection ray
                if(this.options.renderReflections && info.shape.material.reflection > 0)
                {
                    var reflectionRay = this.getReflectionRay(info.position, info.normal, ray.direction);
                    var refl = this.testIntersection(reflectionRay, scene, info.shape);

                    if (refl.isHit && refl.distance > 0){
                        refl.color = this.rayTrace(refl, reflectionRay, scene, depth + 1);
                    } else {
                        refl.color = scene.background.color;
                    }

                    color = Color.blend(
                        color,
                        refl.color,
                        info.shape.material.reflection
                    );
                }

                // Refraction
                /* TODO */
            }

            /* Render shadows and highlights */

            var shadowInfo = new IntersectionInfo();

            if(this.options.renderShadows){
                var shadowRay = new Ray(info.position, v);

                shadowInfo = this.testIntersection(shadowRay, scene, info.shape);
                if(shadowInfo.isHit && shadowInfo.shape != info.shape /*&& shadowInfo.shape.type != 'PLANE'*/){
                    var vA = Color.multiplyScalar(color, 0.5);
                    var dB = (0.5 * Math.pow(shadowInfo.shape.material.transparency, 0.5));
                    color = Color.addScalar(vA,dB);
                }
            }

            // Phong specular highlights
            if(this.options.renderHighlights && !shadowInfo.isHit && info.shape.material.gloss > 0){
                var Lv = Vector.subtract(
                    info.shape.position,
                    light.position
                ).normalize();

                var E = Vector.subtract(
                    scene.camera.position,
                    info.shape.position
                ).normalize();

                var H = Vector.subtract(
                    E,
                    Lv
                ).normalize();

                var glossWeight = Math.pow(Math.max(info.normal.dot(H), 0), shininess);
                color = Color.add(
                    Color.multiplyScalar(light.color, glossWeight),
                    color
                );
            }
        }
        color.limit();
        return color;
    }
}


class Main {

    // Variable used to hold a number that can be used to verify that
    // the scene was ray traced correctly.
    static var checkNumber : number;

    static function renderScene() : void {
        var scene = new Scene();

        scene.camera = new Camera(
            new Vector(0, 0, -15),
            new Vector(-0.2, 0, 5),
            new Vector(0, 1, 0)
        );

        scene.background = new Background(
            new Color(0.5, 0.5, 0.5),
            0.4
        );

        var sphere = new Sphere(
            new Vector(-1.5, 1.5, 2),
            1.5,
            new Solid(
                new Color(0,0.5,0.5),
                0.3,
                0.0,
                0.0,
                2.0
            )
        );

        var sphere1 = new Sphere(
            new Vector(1, 0.25, 1),
            0.5,
            new Solid(
                new Color(0.9,0.9,0.9),
                0.1,
                0.0,
                0.0,
                1.5
            )
        );

        var plane = new Plane(
            new Vector(0.1, 0.9, -0.5).normalize(),
            1.2,
            new Chessboard(
                new Color(1,1,1),
                new Color(0,0,0),
                0.2,
                0.0,
                1.0,
                0.7
            )
        );

        scene.shapes.push(plane);
        scene.shapes.push(sphere);
        scene.shapes.push(sphere1);

        var light = new Light(
            new Vector(5, 10, -1),
            new Color(0.8, 0.8, 0.8)
        );

        var light1 = new Light(
            new Vector(-3, 5, -15),
            new Color(0.8, 0.8, 0.8),
            100
        );

        scene.lights.push(light);
        scene.lights.push(light1);

        var imageWidth = 100; // $F('imageWidth');
        var imageHeight = 100; // $F('imageHeight');
        var pixelSize = "5,5".split(','); //  $F('pixelSize').split(',');
        var renderDiffuse = true; // $F('renderDiffuse');
        var renderShadows = true; // $F('renderShadows');
        var renderHighlights = true; // $F('renderHighlights');
        var renderReflections = true; // $F('renderReflections');
        var rayDepth = 2;//$F('rayDepth');

        var options = new Options();
        options.canvasWidth = imageWidth;
        options.canvasHeight = imageHeight;
        options.pixelWidth = pixelSize[0] as number;
        options.pixelHeight = pixelSize[1] as number;
        options.renderDiffuse = renderDiffuse;
        options.renderHighlights = renderHighlights;
        options.renderShadows = renderShadows;
        options.renderReflections = renderReflections;
        options.rayDepth = rayDepth;

        var raytracer = new Engine(options);

        raytracer.renderScene(scene, null);
    }
}

// vim: set expandtab:
