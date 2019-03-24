module Buildbarn.Browser.Frontend.Route exposing (Route(..), fromUrl, toString)

{-| URL parsing and generation for pages inside this web application.
-}

import Buildbarn.Browser.Frontend.Digest exposing (Digest)
import Url exposing (Url)
import Url.Parser as Parser exposing ((</>))


digestParser : Parser.Parser (Digest -> a) a
digestParser =
    Parser.map Digest (Parser.string </> Parser.string </> Parser.int)


{-| The kind of pages that are provided by this web application.
-}
type Route
    = Action Digest
    | Command Digest
    | Directory Digest
    | Tree Digest (List String)
    | UncachedActionResult Digest
    | Welcome


parser : Parser.Parser (Route -> a) a
parser =
    Parser.oneOf
        [ Parser.map Action (Parser.s "action" </> digestParser)
        , Parser.map Command (Parser.s "command" </> digestParser)
        , Parser.map Directory (Parser.s "directory" </> digestParser)
        , Parser.map Tree (Parser.s "tree" </> digestParser </> Parser.remainder)
        , Parser.map UncachedActionResult (Parser.s "uncached_action_result" </> digestParser)
        , Parser.map Welcome Parser.top
        ]


{-| Converts a URL to a route.
-}
fromUrl : Url -> Maybe Route
fromUrl url =
    { url | path = Maybe.withDefault "" url.fragment, fragment = Nothing }
        |> Parser.parse parser


digestToString : Digest -> String
digestToString digest =
    digest.instance
        ++ "/"
        ++ digest.hash
        ++ "/"
        ++ String.fromInt digest.sizeBytes


{-| Converts a route to a URL string that, when visited, would access
this resource.
-}
toString : Route -> String
toString route =
    case route of
        Action digest ->
            "#action/" ++ digestToString digest

        Command digest ->
            "#command/" ++ digestToString digest

        Directory digest ->
            "#directory/" ++ digestToString digest

        Tree digest subdirectory ->
            "#tree/"
                ++ digestToString digest
                ++ (subdirectory |> List.map ((++) "/") |> String.concat)

        UncachedActionResult digest ->
            "#uncached_action_result/" ++ digestToString digest

        Welcome ->
            "#"
