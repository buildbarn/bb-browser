module Buildbarn.Browser.Frontend.Page.Welcome exposing (view)

import Buildbarn.Browser.Frontend.Page as Page
import Html exposing (a, br, li, p, span, text, ul)
import Html.Attributes exposing (class, href)


{-| List entry on the welcome page that gives a description of a URL
pattern supported by this service.
-}
urlPattern : String -> List (Html.Html msg) -> Html.Html msg
urlPattern pattern description =
    li []
        [ p []
            ([ span [ class "text-monospace" ] [ text pattern ]
             , br [] []
             ]
                ++ description
            )
        ]


view : Page.Page msg
view =
    { title = "Welcome!"
    , bannerColor = "success"
    , body =
        [ p []
            [ text "This page allows you to display objects stored in the Content Addressable Storage (CAS) and Action Cache (AC) as defined by the "
            , a [ href "https://github.com/bazelbuild/remote-apis" ] [ text "Remote Execution API" ]
            , text ". Objects in these data stores have hard to guess identifiers and the Remote Execution API provides no functions for iterating over them.  One may therefore only access this service in a meaningful way by visiting automatically generated URLs pointing to this page. Tools that are part of Buildbarn will generate these URLs where applicable."
            ]
        , p [] [ text "This service supports the following URL schemes:" ]
        , ul []
            [ urlPattern "/action/${instance}/${hash}/${size}/"
                [ text "Displays information about an Action and its associated Command stored in the CAS. If available, displays information about the Action's associated ActionResult stored in the AC."
                ]
            , urlPattern "/build_events/${instance}/${invocation_id}"
                [ text "Extension: displays information about a "
                , a [ href "https://docs.bazel.build/versions/master/build-event-protocol.html" ] [ text "Build Event Stream" ]
                , text ".  The stream is stored in one or more objects in the CAS. An AC entry refers to the objects stored in the CAS."
                ]
            , urlPattern "/#command/${instance}/${hash}/${size}"
                [ text "Displays information about a Command stored in the CAS." ]
            , urlPattern "/#directory/${instance}/${hash}/${size}"
                [ text "Displays information about a Directory (input directory) stored in the CAS." ]
            , urlPattern "/file/${instance}/${hash}/${size}/${filename}"
                [ text "Serves a file stored in the CAS." ]
            , urlPattern "/tree/${instance}/${hash}/${size}/${subdirectory}"
                [ text "Displays information about a Tree (output directory tree) stored in the CAS." ]
            , urlPattern "/uncached_action_result/${instance}/${hash}/${size}/"
                [ text "Extension: displays information about an ActionResult that was not permitted to be stored in the AC, but was stored in the CAS instead. Buildbarn stores ActionResult messages for failed build actions in the CAS." ]
            ]
        ]
    }
