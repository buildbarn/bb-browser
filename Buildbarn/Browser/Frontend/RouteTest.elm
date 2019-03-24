module RouteTest exposing (fromUrl, toString)

import Buildbarn.Browser.Frontend.Digest exposing (Digest)
import Buildbarn.Browser.Frontend.Route as Route exposing (Route(..))
import Expect
import Test exposing (Test)
import Url


vectors : List ( String, Route )
vectors =
    [ ( "action//e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855/0"
      , Action (Digest "" "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" 0)
      )
    , ( "command/debian8/d41d8cd98f00b204e9800998ecf8427e/123"
      , Command (Digest "debian8" "d41d8cd98f00b204e9800998ecf8427e" 123)
      )
    , ( "directory/debian8/d41d8cd98f00b204e9800998ecf8427e/123"
      , Directory (Digest "debian8" "d41d8cd98f00b204e9800998ecf8427e" 123)
      )
    , ( "tree/debian8/d41d8cd98f00b204e9800998ecf8427e/123"
      , Tree (Digest "debian8" "d41d8cd98f00b204e9800998ecf8427e" 123) []
      )
    , ( "tree/debian8/d41d8cd98f00b204e9800998ecf8427e/123/sub/directory"
      , Tree (Digest "debian8" "d41d8cd98f00b204e9800998ecf8427e" 123) [ "sub", "directory" ]
      )
    , ( "uncached_action_result/debian8/d41d8cd98f00b204e9800998ecf8427e/123"
      , UncachedActionResult (Digest "debian8" "d41d8cd98f00b204e9800998ecf8427e" 123)
      )
    , ( ""
      , Welcome
      )
    ]


fromUrl : Test
fromUrl =
    vectors
        |> List.map
            (\( fragment, route ) ->
                Test.test ("Fragment " ++ fragment)
                    (\_ ->
                        Expect.equal
                            (Just route)
                            (Route.fromUrl
                                { protocol = Url.Http
                                , host = "example.com"
                                , port_ = Nothing
                                , path = "/"
                                , query = Nothing
                                , fragment = Just fragment
                                }
                            )
                    )
            )
        |> Test.describe "Route.fromUrl"


toString : Test
toString =
    vectors
        |> List.map
            (\( fragment, route ) ->
                Test.test ("Fragment " ++ fragment)
                    (\_ ->
                        Expect.equal
                            ("#" ++ fragment)
                            (Route.toString route)
                    )
            )
        |> Test.describe "Route.toString"
