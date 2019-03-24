module Buildbarn.Browser.Frontend.Error exposing (Error(..))

import Http
import Parser


type Error
    = ChildMessageMissing
    | Http Http.Error
    | InvalidUtf8
    | Loading
    | Parser (List Parser.DeadEnd)
