module Buildbarn.Browser.Frontend.Page.Command exposing (Model, Msg, init, update, view)

import Bootstrap.Utilities.Spacing exposing (my4)
import Build.Bazel.Remote.Execution.V2.Remote_execution as REv2
import Buildbarn.Browser.Frontend.Api as Api
import Buildbarn.Browser.Frontend.Digest exposing (Digest)
import Buildbarn.Browser.Frontend.Error as Error exposing (Error)
import Buildbarn.Browser.Frontend.Page as Page
import Html exposing (h2, text)
import Http



-- MODEL


type alias Model =
    Result Error REv2.Command


init : Digest -> ( Model, Cmd Msg )
init digest =
    let
        ( e, cmd ) =
            Api.getMessage
                "command"
                GotCommand
                REv2.commandDecoder
                digest
    in
    ( Err e, cmd )



-- UPDATE


type Msg
    = GotCommand Digest (Result Error REv2.Command)


update : Msg -> Model -> ( Model, Cmd Msg )
update (GotCommand _ commandResult) model =
    ( commandResult, Cmd.none )



-- VIEW


view : Model -> Page.Page msg
view model =
    { title = "Command"
    , bannerColor = "secondary"
    , body =
        Page.viewError model <|
            \command ->
                [ Page.viewCommandInfo command
                , h2 [ my4 ] [ text "Output files" ]
                , Page.viewDirectoryListing <|
                    List.map
                        (\path ->
                            Page.viewDirectoryListingEntry
                                "drwxr‑xr‑x"
                                Nothing
                                [ text path, text "/" ]
                        )
                        command.outputDirectories
                        ++ List.map
                            (\path ->
                                Page.viewDirectoryListingEntry
                                    "‑rw‑r‑‑r‑‑"
                                    Nothing
                                    [ text path ]
                            )
                            command.outputFiles
                ]
    }
