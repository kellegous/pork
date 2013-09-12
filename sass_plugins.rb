#!/usr/bin/env ruby

# This is an alternate entry point to the sass command line tool that allows
# us, Monetology, to inject our own functions.

require 'base64'

class Scope
  @options = []
  def self.add(options)
    @options << options
  end
  def self.find(name)
    # first lets check relative to all the files. this is wrong
    # and i need to fix it.

    @options.each { |opt|
      f = File.join(File.dirname(opt[:filename]), name)
      return f if File.exists?(f)
    }

    # next let's look in all the load_paths
    @options.first[:load_paths].each { |p|
      f = File.join(p.root, name)
      return f if File.exists?(f)
    }
    
    #nothing, just return a path that isn't going to work
    return name
  end
end

class Url <Sass::Script::Literal
  def initialize(url)
    super(url)
  end

  def to_s(opts = {})
    "url(\"#{@value}\")"
  end
end

# monkey-patch the _to_tree method in Engine so that we can capture
# the execution context
class Sass::Engine
  alias_method :__to_tree, :_to_tree
  def _to_tree
    Scope.add(@options)
    __to_tree
  end
end

module Sass::Script::Functions
  def datauri(string)
    assert_type string, :String
    name = string.value.downcase
    mime = "application/octet-stream"
    mime = "image/png" if name.end_with?(".png")
    mime = "image/gif" if name.end_with?(".gif")
    mime = "image/jpg" if name.end_with?(".jpg") || name.end_with?(".jpeg")
    file = File.open(Scope.find(string.value), 'rb')
    begin
      # love that gsub at the end? ruby's Base64 adds \n's every
      # 60 characters. Why? I have no ideas. RFC 2045 doesn't
      # say anything about that. It's just the usual ruby community
      # gift!
      data = Base64.encode64(file.read).strip.gsub("\n", "")
      Url.new("data:#{mime};base64,#{data}")
    ensure
      file.close
    end
  end
  declare :datauri, [:string]
end