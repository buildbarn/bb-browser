module Buildbarn.Browser.Frontend.Page.NotFound exposing (view)

import Buildbarn.Browser.Frontend.Page as Page
import Html exposing (p, text)


view : Page.Page msg
view =
    { title = "Page not found"
    , bannerColor = "danger"
    , body = [ p [] [ text "The URL that was requested does not correspond to a page." ] ]
    }
