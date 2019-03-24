module Buildbarn.Browser.Frontend.Page exposing
    ( Page
    , viewChildObjectLink
    , viewCommandInfo
    , viewDirectory
    , viewDirectoryListing
    , viewDirectoryListingEntry
    , viewError
    , viewPage
    )

import Bootstrap.Button as Button
import Bootstrap.CDN as CDN
import Bootstrap.Grid as Grid
import Bootstrap.Navbar as Navbar
import Bootstrap.Table as Table
import Bootstrap.Utilities.Spacing exposing (mb5, my4)
import Browser
import Build.Bazel.Remote.Execution.V2.Remote_execution as REv2
import Buildbarn.Browser.Frontend.Digest as Digest exposing (Digest)
import Buildbarn.Browser.Frontend.Error as Error exposing (Error)
import Buildbarn.Browser.Frontend.Route as Route exposing (Route)
import Buildbarn.Browser.Frontend.Shell as Shell
import Html exposing (a, b, br, div, h1, p, sup, table, td, text, th, tr)
import Html.Attributes exposing (class, href, style)
import Http
import Url.Builder


type alias Page msg =
    { title : String
    , bannerColor : String
    , body : List (Html.Html msg)
    }


viewPage : Page msg -> Browser.Document msg
viewPage contents =
    Browser.Document ("Buildbarn Browser - " ++ contents.title)
        [ Html.nav
            [ class "navbar"
            , class "navbar-dark"
            , class ("bg-" ++ contents.bannerColor)
            ]
            [ a
                [ class "navbar-brand"
                , Route.Welcome |> Route.toString |> href
                ]
                [ text "Buildbarn Browser" ]
            ]
        , Grid.container [ mb5 ] ([ h1 [ my4 ] [ text contents.title ] ] ++ contents.body)
        ]


viewChildObjectLink : (Digest -> Route) -> Digest -> Maybe REv2.DigestMessage -> List (Html.Html msg)
viewChildObjectLink kind parentDigest maybeChildDigest =
    case maybeChildDigest of
        Just (REv2.DigestMessage childDigest) ->
            [ sup []
                [ a
                    [ Digest.getDerived parentDigest childDigest
                        |> kind
                        |> Route.toString
                        |> href
                    ]
                    [ text "*" ]
                ]
            ]

        Nothing ->
            []


viewCommandInfo : REv2.Command -> Html.Html msg
viewCommandInfo command =
    table [ class "table", style "table-layout" "fixed" ] <|
        [ tr []
            [ th [ style "width" "25%" ] [ text "Arguments:" ]
            , td [ class "text-monospace", style "width" "75%", style "overflow-x" "scroll" ] <|
                case command.arguments |> List.map viewShell of
                    first :: rest ->
                        [ div [ style "padding-left" "2em", style "text-indent" "-2em" ] <|
                            b [] [ text first ]
                                :: List.concatMap
                                    (\argument ->
                                        [ text " "
                                        , text argument
                                        ]
                                    )
                                    rest
                        ]

                    [] ->
                        []
            ]
        , tr []
            [ th [ style "width" "25%" ] [ text "Environment variables:" ]
            , command.environmentVariables
                |> List.map
                    (\(REv2.Command_EnvironmentVariableMessage env) ->
                        [ b [] [ text env.name ]
                        , text "="
                        , text <| viewShell env.value
                        ]
                    )
                |> List.intersperse [ br [] [] ]
                |> List.concat
                |> td [ class "text-monospace", style "width" "75%", style "overflow-x" "scroll" ]
            ]
        , tr []
            [ th [ style "width" "25%" ] [ text "Working directory:" ]
            , td [ class "text-monospace", style "width" "75%" ]
                [ text <|
                    if String.isEmpty command.workingDirectory then
                        "."

                    else
                        command.workingDirectory
                ]
            ]
        ]
            ++ (case command.platform of
                    Just (REv2.PlatformMessage platform) ->
                        [ th [ style "width" "25%" ] [ text "Platform properties:" ]
                        , platform.properties
                            |> List.map
                                (\(REv2.Platform_PropertyMessage property) ->
                                    [ b [] [ text property.name ]
                                    , text " = "
                                    , text property.value
                                    ]
                                )
                            |> List.intersperse [ br [] [] ]
                            |> List.concat
                            |> td [ style "width" "75%" ]
                        ]

                    Nothing ->
                        []
               )


viewDirectory : Digest -> REv2.Directory -> List (Html.Html msg)
viewDirectory digest directory =
    [ viewDirectoryListing <|
        List.map
            (\(REv2.DirectoryNodeMessage entry) ->
                viewDirectoryListingEntry
                    "drwxr‑xr‑x"
                    entry.digest
                    [ case entry.digest of
                        Nothing ->
                            text entry.name

                        Just (REv2.DigestMessage childDigest) ->
                            a
                                [ Digest.getDerived digest childDigest
                                    |> Route.Directory
                                    |> Route.toString
                                    |> href
                                ]
                                [ text entry.name ]
                    , text "/"
                    ]
            )
            directory.directories
            ++ List.map
                (\(REv2.FileNodeMessage entry) ->
                    viewDirectoryListingEntry
                        (if entry.isExecutable then
                            "‑r‑xr‑xr‑x"

                         else
                            "‑r‑‑r‑‑r‑‑"
                        )
                        entry.digest
                        [ case entry.digest of
                            Nothing ->
                                text entry.name

                            Just (REv2.DigestMessage childDigest) ->
                                a
                                    [ href <|
                                        let
                                            derivedDigest =
                                                Digest.getDerived digest childDigest
                                        in
                                        Url.Builder.relative
                                            [ "api", "get_file" ]
                                            [ Url.Builder.string "instance" derivedDigest.instance
                                            , Url.Builder.string "hash" derivedDigest.hash
                                            , Url.Builder.int "size_bytes" derivedDigest.sizeBytes
                                            , Url.Builder.string "name" entry.name
                                            ]
                                    ]
                                    [ text entry.name ]
                        ]
                )
                directory.files
            ++ List.map
                (\(REv2.SymlinkNodeMessage entry) ->
                    viewDirectoryListingEntry
                        "lrwxrwxrwx"
                        Nothing
                        [ text entry.name
                        , text " → "
                        , text entry.target
                        ]
                )
                directory.symlinks
    , Button.linkButton
        [ Button.primary
        , Button.attrs
            [ href <|
                Url.Builder.relative
                    [ "api", "get_directory_tarball" ]
                    [ Url.Builder.string "instance" digest.instance
                    , Url.Builder.string "hash" digest.hash
                    , Url.Builder.int "size_bytes" digest.sizeBytes
                    ]
            ]
        ]
        [ text "Download as tarball" ]
    ]


viewDirectoryListingEntry : String -> Maybe REv2.DigestMessage -> List (Html.Html msg) -> Table.Row msg
viewDirectoryListingEntry permissions digest filename =
    Table.tr [ Table.rowAttr <| class "text-monospace" ]
        [ Table.td [] [ text permissions ]
        , Table.td [ Table.cellAttr <| style "text-align" "right" ]
            (case digest of
                Nothing ->
                    []

                Just (REv2.DigestMessage d) ->
                    [ text <| String.fromInt d.sizeBytes ]
            )
        , Table.td [] filename
        ]


viewDirectoryListing : List (Table.Row msg) -> Html.Html msg
viewDirectoryListing entries =
    Table.simpleTable
        ( Table.simpleThead
            [ Table.th [] [ text "Mode" ]
            , Table.th [] [ text "Size" ]
            , Table.th [ Table.cellAttr <| style "width" "100%" ] [ text "Filename" ]
            ]
        , Table.tbody [] entries
        )


{-| Displays a message that the page is still loading, or that a HTTP
error occurred loading it.
-}
viewError : Result Error a -> (a -> List (Html.Html msg)) -> List (Html.Html msg)
viewError result display =
    case result of
        Err error ->
            [ p []
                [ text
                    (case error of
                        Error.ChildMessageMissing ->
                            "Child message missing"

                        Error.Http httpError ->
                            case httpError of
                                Http.BadUrl message ->
                                    "BadURL " ++ message

                                Http.Timeout ->
                                    "Timeout"

                                Http.NetworkError ->
                                    "Network error"

                                Http.BadStatus code ->
                                    "BadCode " ++ String.fromInt code

                                Http.BadBody message ->
                                    "BadBody " ++ message

                        Error.InvalidUtf8 ->
                            "Invalid UTF-8"

                        Error.Loading ->
                            "Loading..."

                        Error.Parser _ ->
                            "Parse error"
                    )
                ]
            ]

        Ok message ->
            display message


viewShell : String -> String
viewShell s =
    s |> Shell.quote |> String.replace "-" "‑"
