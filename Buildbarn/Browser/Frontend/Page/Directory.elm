module Buildbarn.Browser.Frontend.Page.Directory exposing (Model, Msg, init, update, view)

import Build.Bazel.Remote.Execution.V2.Remote_execution as REv2
import Buildbarn.Browser.Frontend.Api as Api
import Buildbarn.Browser.Frontend.Digest exposing (Digest)
import Buildbarn.Browser.Frontend.Error as Error exposing (Error)
import Buildbarn.Browser.Frontend.Page as Page
import Http
import Json.Decode as JD



-- MODEL


type alias Model =
    Result Error
        { digest : Digest
        , directory : REv2.Directory
        }


init : Digest -> ( Model, Cmd Msg )
init digest =
    let
        ( e, cmd ) =
            Api.getMessage
                "directory"
                GotDirectory
                REv2.directoryDecoder
                digest
    in
    ( Err e, cmd )



-- UPDATE


type Msg
    = GotDirectory Digest (Result Error REv2.Directory)


update : Msg -> Model -> ( Model, Cmd Msg )
update (GotDirectory digest directoryResult) model =
    ( Result.map
        (\directory -> { digest = digest, directory = directory })
        directoryResult
    , Cmd.none
    )



-- VIEW


view : Model -> Page.Page msg
view model =
    { title = "Input directory"
    , bannerColor = "secondary"
    , body =
        Page.viewError model <|
            \message -> Page.viewDirectory message.digest message.directory
    }
