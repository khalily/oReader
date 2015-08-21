'use strict';

var oReaderApp = angular.module('oReaderApp');

oReaderApp.service('auth', ['$http', '$q', '$rootScope', function ($http, $q, $rootScope) {

    var self = this;
    self.profile = null;
    self.isAuthenticated = false;
    self.isCacheTokenInvalid = false;

    function request(url, email_or_token, password) {
        return {
            method: 'GET',
            url: url,
            headers: {
                Accept: 'application/json',
                Authorization: 'Basic ' + btoa(email_or_token + ':' + password)
            }
        };
    }

    self.signin = function (options, successCallback, errorCallback) {
        var req = request('/api/get_token', options.email, options.password);
        $http(req).
        then(function(response) {
                var token = response.data.token;
                if (token) {
                    self.isAuthenticated = true;
                    self.profile = response.data.profile;
                    successCallback(token);
                }

            }, function(error) {
                console.log(error.message);
                errorCallback();
            }
        );

    };
    
    self.signout = function () {
        self.isAuthenticated = false;
        self.profile = null;
    };

    self.authenticate = function () {
        var req = request('/api/login', '', '');
        var deferred = $q.defer();
        $http(req)
            .then(function (response) {
                self.profile = response.data.profile;
                self.isAuthenticated = true;
                deferred.resolve();
            }, function () {
                self.isAuthenticated = false;
                deferred.reject();
            });
        return deferred.promise;
    }

}]);

oReaderApp.factory('tokenInjector', ['store', function (store) {
    return {
        'request': function(config) {
            var token = store.get('token');
            if (token && config.url != '/api/get_token') {
                config.headers['Authorization'] = 'Basic ' + btoa(token + ':' + '');
            }
            return config;
        }
    }
}]);

oReaderApp.factory('tokenInvalidInterceptor', ['$q', '$location', '$rootScope', 'store',
    function ($q, $location, $rootScope, store) {
    return {
        'response': function (response) {
            if (response.status == 401) {
                console.log('401 response', response);
                if (store.get('token'))
                    $rootScope.$broadcast('unauth_token');
                else
                    $rootScope.$broadcast('unauth');
                return $q.reject();
            }
            return $q.resolve(response);
        },

        'responseError': function (rejection) {
            if (rejection.status == 401) {
                console.log('401 error', rejection);
                if (store.get('token'))
                    $rootScope.$broadcast('unauth_token');
                else
                    $rootScope.$broadcast('unauth');
            }
            return $q.reject(rejection);
        }
    }
}]);

oReaderApp.factory('Feed', ['$resource', function ($resource) {
    return $resource('/api/feeds/:id', {id: '@id'});
}]);

oReaderApp.factory('feeds', ['Feed', function(Feed) {
    return Feed.query();
}]);

oReaderApp.factory('addFeed', ['feeds', function(feeds) {
    return function (feed, successCallback, errorCallback) {
        if (!feed.id) {
            //feeds.push(feed);
        }
        return feed.$save(successCallback, errorCallback);
    }
}]);

oReaderApp.factory('Story', ['$resource', function ($resource) {
    return $resource('/api/feeds/:feed_id/story/:story_id', {feed_id: '@feed_id', story_id: '@story_id'});
}]);

oReaderApp.factory('stories', ['Story', function (Story) {
    return function(feed_id) {
        return Story.query({feed_id: feed_id})
    }
}]);

oReaderApp.factory('User', ['$resource', function ($resource) {
    return $resource('/api/users/:user_id', {user_id: '@user_id'})
}]);

oReaderApp.factory('get_user', ['User', function (User) {
    return function (user_id) {
        return User.get({user_id: user_id})
    }
}]);
