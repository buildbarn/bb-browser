module Buildbarn.Browser.Frontend.Digest exposing (Digest, getDerived)

import Build.Bazel.Remote.Execution.V2.Remote_execution as REv2


type alias Digest =
    { instance : String
    , hash : String
    , sizeBytes : Int
    }


getDerived : Digest -> REv2.Digest -> Digest
getDerived parent child =
    { instance = parent.instance
    , hash = child.hash
    , sizeBytes = child.sizeBytes
    }
