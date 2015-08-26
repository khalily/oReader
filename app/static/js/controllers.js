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

oReaderApp.controller('FeedCtrl', ['$scope', 'Restangular', '$location', function ($scope, Restangular, $location) {
    Restangular.one('feeds/').getList().then(function (feeds) {
        $scope.feeds = feeds;
        $scope.currentFeed = $scope.feeds[0];
    });

    $scope.subscription = function () {
        $location.path('/subscription');
    };

    $scope.setCurrentFeed = function (feed) {
        $scope.currentFeed = feed;
    };

    function scrollIntoView(element, container) {
        console.log(element, container);
        var containerTop = container.scrollTop();
        var containerBottom = containerTop + container.height();
        var elemTop = element.offset().top;
        var elemBottom = elemTop + element.height();
        console.log(containerTop, containerBottom, elemTop, elemBottom);
        //if (elemTop < containerTop) {
        container.scrollTop(elemTop + containerTop - 50);
        //} else if (elemBottom > containerBottom) {
        //    container.scrollTop(elemBottom - container.height());
        //}
    }

    $scope.updateStates = function (i, item) {
        if ($scope.currentItem == item) {
            $scope.currentItem = null;
        } else {
            $scope.currentItem = item;
            setTimeout(function() {scrollIntoView($('#storydiv' + i), $('.story-outline'));});
        }
    };
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
