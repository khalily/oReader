'use strict';

var oReaderApp = angular.module('oReaderApp');

oReaderApp.controller('RootCtrl', ['$scope', 'auth','$location', 'store', 'get_user', '$rootScope',
    function ($scope, auth, $location, store, get_user, $rootScope) {

    $scope.auth = auth;
    $scope.token = "";

    $scope.$watch('auth.isAuthenticated', function(isAuthenticated) {
        if (isAuthenticated) {
            $rootScope.loggedUser = auth.profile;
            $location.path('/feeds');
        } else {
            $rootScope.loggedUser = null;
        }
    });

    $scope.stories = [];
}]);

oReaderApp.controller('LoginCtrl', ['auth', 'store', '$location', '$scope', '$rootScope',
    function(auth, store, $location, $scope, $rootScope) {
    $scope.user = {
        email: '',
        password: ''
    };
    $scope.hitMsg = '';

    function onLoginSuccess(token) {
        $scope.loading = false;
        store.set('token', token);
        $scope.hitMsg = '登录成功';
        $('#login_dialog').modal('hide');
        $scope.hitMsg = '';
        $location.path('/');
    }

    function onLoginFailed() {
       $scope.loading = false;
       $scope.hitMsg = '邮箱或密码不正确';
    }

    $scope.submit = function() {
        console.log('login');
        $scope.hitMsg = '';
        $scope.loading = true;
        auth.signin({
            email: $scope.user.email,
            password: $scope.user.password
        }, onLoginSuccess, onLoginFailed);
    };

    $scope.addSubscription = function() {
        alert('addSubscription');
        $location.path('/addSubscription');
    };

    $rootScope.$on('unauth_token', function () {
        store.remove('token');
        $scope.hitMsg = '缓存已失效，请重新登录';
        $('#login_dialog').modal('show');
    });

    $rootScope.$on('unauth', function () {
        store.remove('token');
        $scope.hitMsg = '请登录';
        $('#login_dialog').modal('show');
    });

}]);

oReaderApp.controller('LogoutCtrl', ["auth", "$location", "store", function (auth, $location, store) {
    console.log('logout');
    auth.signout();
    store.remove('token');
    $location.path('/');
}]);

oReaderApp.controller('AboutCtrl', ['auth', function(auth) {
    console.log('about ', auth.isAuthenticated);
}]);

oReaderApp.controller('FeedCtrl', ['$scope', 'Restangular', function ($scope, Restangular) {
    $scope.feeds = Restangular.one('feeds/').getList().$object;
}]);

oReaderApp.controller('SubscriptionCtrl', ['$scope', 'Restangular', '$location',
    function ($scope, Restangular, $location) {
    $scope.rss_url = "";
    $scope.submit = function () {
        alert('submit');
        if ($scope.rss_url) {
            Restangular.one('feeds/').customPOST({url: $scope.rss_url}).then(function(feed) {
                $scope.feed = feed;
                //$location.path('/feeds');
            }, function (error) {
                alert(error);
            });
        }
    };
}]);
