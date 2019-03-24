module Main exposing (main)

import Browser
import Browser.Navigation as Navigation
import Buildbarn.Browser.Frontend.Page as Page
import Buildbarn.Browser.Frontend.Page.Action as PageAction
import Buildbarn.Browser.Frontend.Page.Command as PageCommand
import Buildbarn.Browser.Frontend.Page.Directory as PageDirectory
import Buildbarn.Browser.Frontend.Page.NotFound as PageNotFound
import Buildbarn.Browser.Frontend.Page.Tree as PageTree
import Buildbarn.Browser.Frontend.Page.Welcome as PageWelcome
import Buildbarn.Browser.Frontend.Route as Route
import Platform.Cmd
import Platform.Sub
import Url exposing (Url)



-- MODEL


type CurrentPage
    = Action PageAction.Model
    | Command PageCommand.Model
    | Directory PageDirectory.Model
    | NotFound
    | Tree PageTree.Model
    | Welcome


type alias Model =
    { currentPage : CurrentPage
    , navigationKey : Navigation.Key
    }


type alias Flags =
    {}


init : Flags -> Url -> Navigation.Key -> ( Model, Cmd Msg )
init flags url navigationKey =
    changeRouteTo (Route.fromUrl url)
        { currentPage = NotFound
        , navigationKey = navigationKey
        }


changeRouteTo : Maybe Route.Route -> Model -> ( Model, Cmd Msg )
changeRouteTo maybeRoute model =
    case maybeRoute of
        Nothing ->
            ( { model | currentPage = NotFound }, Cmd.none )

        Just (Route.Action digest) ->
            PageAction.initCached digest
                |> updateWith Action GotActionMsg model

        Just (Route.Command digest) ->
            PageCommand.init digest
                |> updateWith Command GotCommandMsg model

        Just (Route.Directory digest) ->
            PageDirectory.init digest
                |> updateWith Directory GotDirectoryMsg model

        Just (Route.Tree digest path) ->
            PageTree.init digest path
                |> updateWith Tree GotTreeMsg model

        Just (Route.UncachedActionResult digest) ->
            PageAction.initUncached digest
                |> updateWith Action GotActionMsg model

        Just Route.Welcome ->
            ( { model | currentPage = Welcome }, Cmd.none )



-- UPDATE


type Msg
    = ChangedUrl Url
    | ClickedLink Browser.UrlRequest
    | GotActionMsg PageAction.Msg
    | GotCommandMsg PageCommand.Msg
    | GotDirectoryMsg PageDirectory.Msg
    | GotTreeMsg PageTree.Msg


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case ( msg, model.currentPage ) of
        ( ChangedUrl url, _ ) ->
            changeRouteTo (Route.fromUrl url) model

        ( ClickedLink urlRequest, _ ) ->
            case urlRequest of
                Browser.Internal url ->
                    if url.path /= "" then
                        -- Internal link, but not to this application.
                        ( model
                        , Navigation.load <| Url.toString url
                        )

                    else
                        -- Internal link inside this application.
                        ( model
                        , Navigation.pushUrl model.navigationKey (Url.toString url)
                        )

                Browser.External href ->
                    ( model
                    , Navigation.load href
                    )

        ( GotActionMsg subMsg, Action subModel ) ->
            PageAction.update subMsg subModel
                |> updateWith Action GotActionMsg model

        ( GotCommandMsg subMsg, Command subModel ) ->
            PageCommand.update subMsg subModel
                |> updateWith Command GotCommandMsg model

        ( GotDirectoryMsg subMsg, Directory subModel ) ->
            PageDirectory.update subMsg subModel
                |> updateWith Directory GotDirectoryMsg model

        ( GotTreeMsg subMsg, Tree subModel ) ->
            PageTree.update subMsg subModel
                |> updateWith Tree GotTreeMsg model

        -- Ignore invalid message/model pairs.
        ( _, _ ) ->
            ( model, Cmd.none )


updateWith : (subModel -> CurrentPage) -> (subMsg -> Msg) -> Model -> ( subModel, Cmd subMsg ) -> ( Model, Cmd Msg )
updateWith toModel toMsg model ( subModel, subCmd ) =
    ( { model | currentPage = toModel subModel }
    , Cmd.map toMsg subCmd
    )



-- VIEW


view : Model -> Browser.Document Msg
view model =
    case model.currentPage of
        Action subModel ->
            Page.viewPage <| PageAction.view subModel

        Command subModel ->
            Page.viewPage <| PageCommand.view subModel

        Directory subModel ->
            Page.viewPage <| PageDirectory.view subModel

        NotFound ->
            Page.viewPage PageNotFound.view

        Tree subModel ->
            Page.viewPage <| PageTree.view subModel

        Welcome ->
            Page.viewPage PageWelcome.view



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.none



-- MAIN


main =
    Browser.application
        { init = init
        , onUrlChange = ChangedUrl
        , onUrlRequest = ClickedLink
        , subscriptions = subscriptions
        , update = update
        , view = view
        }
