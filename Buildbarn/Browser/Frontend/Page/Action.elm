module Buildbarn.Browser.Frontend.Page.Action exposing
    ( Model
    , Msg
    , initCached
    , initUncached
    , update
    , view
    )

import Bootstrap.Badge as Badge
import Bootstrap.Utilities.Spacing exposing (my4)
import Build.Bazel.Remote.Execution.V2.Remote_execution as REv2
import Buildbarn.Browser.Frontend.Api as Api
import Buildbarn.Browser.Frontend.Digest as Digest exposing (Digest)
import Buildbarn.Browser.Frontend.Error as Error exposing (Error)
import Buildbarn.Browser.Frontend.Page as Page
import Buildbarn.Browser.Frontend.Route as Route
import Buildbarn.Browser.Frontend.Terminal as Terminal
import Bytes exposing (Bytes)
import Bytes.Decode as BD
import Google.Protobuf.Duration as Duration
import Html exposing (Html, a, div, h2, p, span, table, td, text, th, tr)
import Html.Attributes exposing (class, href, style)
import Http
import Json.Decode as JD
import Parser
import Pkg.Proto.Cas.Cas as Cas
import Url.Builder



-- MODEL


type alias Model =
    { action : Result Error ActionModel
    , actionResult : Result Error ActionResultModel
    , baseDigest : Digest
    }


initCached : Digest -> ( Model, Cmd Msg )
initCached digest =
    let
        ( action, actionCmd ) =
            Api.getMessage
                "action"
                GotAction
                REv2.actionDecoder
                digest

        ( actionResult, actionResultCmd ) =
            Api.getMessage
                "action_result"
                GotActionResult
                REv2.actionResultDecoder
                digest
    in
    ( { action = Err action
      , actionResult = Err action
      , baseDigest = digest
      }
    , Cmd.batch [ actionCmd, actionResultCmd ]
    )


initUncached : Digest -> ( Model, Cmd Msg )
initUncached digest =
    let
        ( e, cmd ) =
            Api.getMessage
                "uncached_action_result"
                GotUncachedActionResult
                Cas.uncachedActionResultDecoder
                digest
    in
    ( { action = Err e
      , actionResult = Err e
      , baseDigest = digest
      }
    , cmd
    )


type alias ActionModel =
    { data : REv2.Action
    , command : Result Error REv2.Command
    , inputRoot : Result Error REv2.Directory
    }


initActionModel : Digest -> REv2.Action -> ( ActionModel, Cmd Msg )
initActionModel digest action =
    let
        ( commandError, commandCmds ) =
            Api.getChildMessage
                "command"
                GotCommand
                REv2.commandDecoder
                action.commandDigest
                digest

        ( inputRootError, inputRootCmds ) =
            Api.getChildMessage
                "directory"
                GotInputRoot
                REv2.directoryDecoder
                action.inputRootDigest
                digest
    in
    ( { data = action
      , command = Err commandError
      , inputRoot = Err inputRootError
      }
    , Cmd.batch [ commandCmds, inputRootCmds ]
    )


type alias StreamModel =
    List Terminal.FormattedTextFragment


initStreamModelString : String -> ( Result Error StreamModel, Cmd Msg )
initStreamModelString str =
    ( Parser.run (Terminal.formattedTextFragments Terminal.defaultAttributes) str
        |> Result.map .textFragments
        |> Result.mapError Error.Parser
    , Cmd.none
    )


initStreamModelRaw : Bytes -> ( Result Error StreamModel, Cmd Msg )
initStreamModelRaw raw =
    case BD.decode (BD.string (Bytes.width raw)) raw of
        Just str ->
            initStreamModelString str

        _ ->
            ( Err Error.InvalidUtf8, Cmd.none )


initStreamModelDigest : REv2.Digest -> (Result Http.Error String -> Msg) -> ( Result Error StreamModel, Cmd Msg )
initStreamModelDigest digest toMsg =
    -- TODO: Fail if the size is too big.
    -- TODO: Fill in the right instance name.
    ( Err Error.Loading
    , Http.get
        { url =
            Url.Builder.relative
                [ "api", "get_file" ]
                [ Url.Builder.string "instance" "TODO"
                , Url.Builder.string "hash" digest.hash
                , Url.Builder.int "size_bytes" digest.sizeBytes
                ]
        , expect = Http.expectString toMsg
        }
    )


initStreamModel : Bytes -> Maybe REv2.DigestMessage -> (Result Http.Error String -> Msg) -> ( Result Error StreamModel, Cmd Msg )
initStreamModel raw maybeDigest toMsg =
    if Bytes.width raw > 0 then
        initStreamModelRaw raw

    else
        case maybeDigest of
            Just (REv2.DigestMessage digest) ->
                if digest.sizeBytes > 0 then
                    initStreamModelDigest digest toMsg

                else
                    initStreamModelRaw raw

            _ ->
                initStreamModelRaw raw


type alias ActionResultModel =
    { data : REv2.ActionResult
    , stderr : Result Error StreamModel
    , stdout : Result Error StreamModel
    }


initActionResultModel : REv2.ActionResult -> ( ActionResultModel, Cmd Msg )
initActionResultModel actionResult =
    let
        ( stdout, stdoutCmds ) =
            initStreamModel actionResult.stdoutRaw actionResult.stdoutDigest GotStdout

        ( stderr, stderrCmds ) =
            initStreamModel actionResult.stderrRaw actionResult.stderrDigest GotStderr
    in
    ( { data = actionResult
      , stdout = stdout
      , stderr = stderr
      }
    , Cmd.batch [ stdoutCmds, stderrCmds ]
    )



-- UPDATE


type Msg
    = GotAction Digest (Result Error REv2.Action)
    | GotActionResult Digest (Result Error REv2.ActionResult)
    | GotCommand Digest (Result Error REv2.Command)
    | GotInputRoot Digest (Result Error REv2.Directory)
    | GotStderr (Result Http.Error String)
    | GotStdout (Result Http.Error String)
    | GotUncachedActionResult Digest (Result Error Cas.UncachedActionResult)


updateStream : Model -> ((Result Error StreamModel -> Result Error StreamModel) -> ActionResultModel -> ActionResultModel) -> Result Http.Error String -> ( Model, Cmd Msg )
updateStream model mapStream body =
    ( model
        |> (mapFieldActionResult <|
                Result.map <|
                    mapStream <|
                        \_ ->
                            Result.mapError Error.Http body
                                |> Result.andThen
                                    (\bodyString ->
                                        Parser.run (Terminal.formattedTextFragments Terminal.defaultAttributes) bodyString
                                            |> Result.map .textFragments
                                            |> Result.mapError Error.Parser
                                    )
           )
    , Cmd.none
    )


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        GotAction actionDigest maybeAction ->
            case maybeAction of
                Err e ->
                    ( { model | action = Err e }, Cmd.none )

                Ok action ->
                    let
                        ( actionModel, cmds ) =
                            initActionModel actionDigest action
                    in
                    ( { model | action = Ok actionModel }, cmds )

        GotActionResult _ maybeActionResult ->
            case maybeActionResult of
                Err e ->
                    ( { model | actionResult = Err e }, Cmd.none )

                Ok actionResult ->
                    let
                        ( actionResultModel, cmds ) =
                            initActionResultModel actionResult
                    in
                    ( { model | actionResult = Ok actionResultModel }, cmds )

        GotCommand _ command ->
            ( model
                |> (mapFieldAction <|
                        Result.map <|
                            mapFieldCommand <|
                                \_ -> command
                   )
            , Cmd.none
            )

        GotInputRoot _ directory ->
            ( model
                |> (mapFieldAction <|
                        Result.map <|
                            mapFieldInputRoot <|
                                \_ -> directory
                   )
            , Cmd.none
            )

        GotStderr body ->
            updateStream model mapFieldStderr body

        GotStdout body ->
            updateStream model mapFieldStdout body

        GotUncachedActionResult uncachedActionResultDigest maybeUncachedActionResult ->
            case maybeUncachedActionResult of
                Err e ->
                    ( { model | action = Err e, actionResult = Err e }
                    , Cmd.none
                    )

                Ok uncachedActionResult ->
                    let
                        ( action, actionCmd ) =
                            Api.getChildMessage
                                "action"
                                GotAction
                                REv2.actionDecoder
                                uncachedActionResult.actionDigest
                                uncachedActionResultDigest

                        ( actionResult, actionResultCmd ) =
                            case uncachedActionResult.actionResult of
                                Nothing ->
                                    ( Err Error.ChildMessageMissing, Cmd.none )

                                Just (REv2.ActionResultMessage actionResultMessage) ->
                                    initActionResultModel actionResultMessage
                                        |> (\( m, c ) -> ( Ok m, c ))
                    in
                    ( { model | action = Err action, actionResult = actionResult }
                    , Cmd.batch [ actionCmd, actionResultCmd ]
                    )


mapFieldAction : (a -> a) -> { b | action : a } -> { b | action : a }
mapFieldAction updater record =
    { record | action = updater record.action }


mapFieldActionResult : (a -> a) -> { b | actionResult : a } -> { b | actionResult : a }
mapFieldActionResult updater record =
    { record | actionResult = updater record.actionResult }


mapFieldCommand : (a -> a) -> { b | command : a } -> { b | command : a }
mapFieldCommand updater record =
    { record | command = updater record.command }


mapFieldInputRoot : (a -> a) -> { b | inputRoot : a } -> { b | inputRoot : a }
mapFieldInputRoot updater record =
    { record | inputRoot = updater record.inputRoot }


mapFieldStderr : (a -> a) -> { b | stderr : a } -> { b | stderr : a }
mapFieldStderr updater record =
    { record | stderr = updater record.stderr }


mapFieldStdout : (a -> a) -> { b | stdout : a } -> { b | stdout : a }
mapFieldStdout updater record =
    { record | stdout = updater record.stdout }



-- VIEW


view : Model -> Page.Page msg
view model =
    { title = "Action"
    , bannerColor =
        case model.actionResult of
            Ok actionResult ->
                if actionResult.data.exitCode == 0 then
                    "success"

                else
                    "danger"

            _ ->
                "secondary"
    , body =
        (Page.viewError model.action <|
            \actionModel ->
                [ table [ class "table", style "table-layout" "fixed" ] <|
                    [ tr []
                        [ th [ style "width" "25%" ] [ text "Timeout:" ]
                        , td [ style "width" "75%" ] <|
                            case actionModel.data.timeout of
                                Just (Duration.DurationMessage timeout) ->
                                    [ text <| String.fromInt timeout.seconds
                                    , text " seconds "
                                    ]

                                _ ->
                                    [ text "∞" ]
                        ]
                    , tr []
                        [ th [ style "width" "25%" ] [ text "Do not cache:" ]
                        , td [ style "width" "75%" ]
                            [ text <|
                                if actionModel.data.doNotCache then
                                    "yes"

                                else
                                    "no"
                            ]
                        ]
                    ]
                , h2 [ my4 ]
                    (text "Command"
                        :: Page.viewChildObjectLink
                            Route.Command
                            model.baseDigest
                            actionModel.data.commandDigest
                    )
                ]
                    ++ (Page.viewError actionModel.command <|
                            \command -> [ Page.viewCommandInfo command ]
                       )
        )
            ++ [ h2 [ my4 ] [ text "Result " ] ]
            ++ (Page.viewError model.actionResult <|
                    \actionResult ->
                        [ table [ class "table", style "table-layout" "fixed" ] <|
                            [ tr []
                                [ th [ style "width" "25%" ] [ text "Exit code:" ]
                                , td [ style "width" "75%" ]
                                    [ text <| String.fromInt actionResult.data.exitCode
                                    , text " "
                                    , if actionResult.data.exitCode == 0 then
                                        Badge.badgeSuccess [] [ text "Success " ]

                                      else
                                        Badge.badgeDanger [] [ text "Failure " ]
                                    ]
                                ]
                            ]
                                ++ viewStream "Standard output" actionResult.stdout
                                ++ viewStream "Standard error" actionResult.stderr
                        ]
               )
            ++ (case model.action of
                    Ok actionModel ->
                        h2 [ my4 ]
                            (text "Input files"
                                :: Page.viewChildObjectLink
                                    Route.Directory
                                    model.baseDigest
                                    actionModel.data.inputRootDigest
                            )
                            :: (Page.viewError actionModel.inputRoot <|
                                    -- TODO: Use the right digest.
                                    Page.viewDirectory { instance = "", hash = "", sizeBytes = 0 }
                               )

                    Err _ ->
                        []
               )
            ++ (case
                    ( model.actionResult
                    , model.action
                        |> Result.andThen (\actionModel -> actionModel.command)
                    )
                of
                    ( Err _, Err _ ) ->
                        []

                    ( maybeActionResult, maybeCommand ) ->
                        [ h2 [ my4 ] [ text "Output files" ]
                        , Page.viewDirectoryListing <|
                            (case maybeActionResult of
                                Ok actionResult ->
                                    List.map
                                        (\(REv2.OutputDirectoryMessage entry) ->
                                            Page.viewDirectoryListingEntry
                                                "drwxrwxrwx"
                                                entry.treeDigest
                                                [ case entry.treeDigest of
                                                    Nothing ->
                                                        span [ class "text-success" ] [ text entry.path ]

                                                    Just (REv2.DigestMessage childDigest) ->
                                                        a
                                                            [ class "text-success"
                                                            , Route.Tree (Digest.getDerived model.baseDigest childDigest) []
                                                                |> Route.toString
                                                                |> href
                                                            ]
                                                            [ text entry.path ]
                                                , text "/"
                                                ]
                                        )
                                        actionResult.data.outputDirectories
                                        ++ List.map
                                            (\(REv2.OutputSymlinkMessage entry) ->
                                                Page.viewDirectoryListingEntry
                                                    "lrwxrwxrwx"
                                                    Nothing
                                                    [ span
                                                        [ class "text-success" ]
                                                        [ text entry.path ]
                                                    , text " → "
                                                    , text entry.target
                                                    , text "/"
                                                    ]
                                            )
                                            actionResult.data.outputDirectorySymlinks
                                        ++ List.map
                                            (\(REv2.OutputFileMessage entry) ->
                                                Page.viewDirectoryListingEntry
                                                    (if entry.isExecutable then
                                                        "‑rwxr‑xr‑x"

                                                     else
                                                        "‑rw‑r‑‑r‑‑"
                                                    )
                                                    entry.digest
                                                    [ case entry.digest of
                                                        Nothing ->
                                                            span [ class "text-success" ] [ text entry.path ]

                                                        Just (REv2.DigestMessage childDigest) ->
                                                            -- TODO: Link to the file.
                                                            span [ class "text-success" ] [ text entry.path ]
                                                    ]
                                            )
                                            actionResult.data.outputFiles
                                        ++ List.map
                                            (\(REv2.OutputSymlinkMessage entry) ->
                                                Page.viewDirectoryListingEntry
                                                    "lrwxrwxrwx"
                                                    Nothing
                                                    [ span
                                                        [ class "text-success" ]
                                                        [ text entry.path ]
                                                    , text " → "
                                                    , text entry.target
                                                    ]
                                            )
                                            actionResult.data.outputFileSymlinks

                                Err _ ->
                                    []
                            )
                                ++ (case maybeCommand of
                                        Ok command ->
                                            []

                                        Err _ ->
                                            []
                                   )
                        ]
               )
    }


convertColor : Terminal.Color -> Bool -> String
convertColor color bold =
    if bold then
        case color of
            Terminal.Black ->
                "#7f7f7f"

            Terminal.Red ->
                "#ff0000"

            Terminal.Green ->
                "#00ff00"

            Terminal.Brown ->
                "#ffff00"

            Terminal.Blue ->
                "#5c5cff"

            Terminal.Magenta ->
                "#ff00ff"

            Terminal.Cyan ->
                "#00ffff"

            Terminal.White ->
                "#ffffff"

    else
        case color of
            Terminal.Black ->
                "#000000"

            Terminal.Red ->
                "#cd0000"

            Terminal.Green ->
                "#00cd00"

            Terminal.Brown ->
                "#cdcd00"

            Terminal.Blue ->
                "#0000ee"

            Terminal.Magenta ->
                "#cd00cd"

            Terminal.Cyan ->
                "#00cdcd"

            Terminal.White ->
                "#e5e5e5"


convertTerminalAttributes : Terminal.Attributes -> List (Html.Attribute msg)
convertTerminalAttributes attributes =
    (if attributes.bold then
        [ style "font-weight" "bold" ]

     else
        []
    )
        ++ (if attributes.underline then
                [ style "text-decoration" "underline" ]

            else
                []
           )
        ++ (let
                foreground =
                    convertColor
                        (Maybe.withDefault Terminal.White attributes.foreground)
                        attributes.bold

                background =
                    convertColor
                        (Maybe.withDefault Terminal.Black attributes.background)
                        False
            in
            [ style "color" <|
                if attributes.reverse then
                    background

                else
                    foreground
            , style "background-color" <|
                if attributes.reverse then
                    foreground

                else
                    background
            ]
           )


viewStream : String -> Result Error StreamModel -> List (Html msg)
viewStream name model =
    case model of
        Ok [] ->
            -- Omit empty log output.
            []

        _ ->
            [ tr []
                [ th [ style "width" "25%" ]
                    (text name
                        -- TODO: Enable this.
                        {- :: Page.viewChildObjectLink
                           "log"
                           model.baseDigest
                        -}
                        :: [ text ":" ]
                    )
                , td [ style "width" "75%" ] <|
                    Page.viewError model <|
                        \stream ->
                            [ stream
                                |> List.map
                                    (\( attributes, body ) ->
                                        span (convertTerminalAttributes attributes) [ text body ]
                                    )
                                |> div
                                    [ class "text-monospace"
                                    , style "background-color" "#000000"
                                    , style "border-radius" "5px"
                                    , style "font-size" "12px"
                                    , style "line-height" "20px"
                                    , style "overflow-wrap" "break-word"
                                    , style "padding" "14px 18px"
                                    , style "white-space" "pre-wrap"
                                    , style "word-break" "break-word"
                                    ]
                            ]
                ]
            ]
