package ssh

import (
	"reflect"
	"testing"
)

func TestBuildConfig(t *testing.T) {
	cases := []struct {
		config   *Config
		defaults *Config
		result   *Config
	}{
		// All defaults
		{
			nil,
			nil,
			&Config{
				Port:              Port,
				User:              User,
				ConnectionTimeout: ConnectionTimeout,
				RetryTimeout:      RetryTimeout,
				RetryInterval:     RetryInterval,
			},
		},

		// PrivateKey
		{
			&Config{
				PrivateKey: "foo",
			},
			nil,
			&Config{
				PrivateKey:        "foo",
				Port:              Port,
				User:              User,
				ConnectionTimeout: ConnectionTimeout,
				RetryTimeout:      RetryTimeout,
				RetryInterval:     RetryInterval,
			},
		},
		{
			&Config{
				PrivateKey: "foo",
			},
			&Config{
				PrivateKey: "bar",
			},
			&Config{
				PrivateKey:        "foo",
				Port:              Port,
				User:              User,
				ConnectionTimeout: ConnectionTimeout,
				RetryTimeout:      RetryTimeout,
				RetryInterval:     RetryInterval,
			},
		},
		{
			nil,
			&Config{
				PrivateKey: "bar",
			},
			&Config{
				PrivateKey:        "bar",
				Port:              Port,
				User:              User,
				ConnectionTimeout: ConnectionTimeout,
				RetryTimeout:      RetryTimeout,
				RetryInterval:     RetryInterval,
			},
		},

		// User
		{
			&Config{
				User: "foo",
			},
			nil,
			&Config{
				User:              "foo",
				Port:              Port,
				ConnectionTimeout: ConnectionTimeout,
				RetryTimeout:      RetryTimeout,
				RetryInterval:     RetryInterval,
			},
		},
		{
			&Config{
				User: "foo",
			},
			&Config{
				User: "bar",
			},
			&Config{
				User:              "foo",
				Port:              Port,
				ConnectionTimeout: ConnectionTimeout,
				RetryTimeout:      RetryTimeout,
				RetryInterval:     RetryInterval,
			},
		},
		{
			nil,
			&Config{
				User: "bar",
			},
			&Config{
				User:              "bar",
				Port:              Port,
				ConnectionTimeout: ConnectionTimeout,
				RetryTimeout:      RetryTimeout,
				RetryInterval:     RetryInterval,
			},
		},

		// ConnectionTimeout
		{
			&Config{
				ConnectionTimeout: "foo",
			},
			nil,
			&Config{
				ConnectionTimeout: "foo",
				Port:              Port,
				User:              User,
				RetryTimeout:      RetryTimeout,
				RetryInterval:     RetryInterval,
			},
		},
		{
			&Config{
				ConnectionTimeout: "foo",
			},
			&Config{
				ConnectionTimeout: "bar",
			},
			&Config{
				ConnectionTimeout: "foo",
				Port:              Port,
				User:              User,
				RetryTimeout:      RetryTimeout,
				RetryInterval:     RetryInterval,
			},
		},
		{
			nil,
			&Config{
				ConnectionTimeout: "bar",
			},
			&Config{
				ConnectionTimeout: "bar",
				Port:              Port,
				User:              User,
				RetryTimeout:      RetryTimeout,
				RetryInterval:     RetryInterval,
			},
		},

		// Port
		{
			&Config{
				Port: 33,
			},
			nil,
			&Config{
				ConnectionTimeout: ConnectionTimeout,
				Port:              33,
				User:              User,
				RetryTimeout:      RetryTimeout,
				RetryInterval:     RetryInterval,
			},
		},
		{
			&Config{
				Port: 33,
			},
			&Config{
				Port: 44,
			},
			&Config{
				ConnectionTimeout: ConnectionTimeout,
				Port:              33,
				User:              User,
				RetryTimeout:      RetryTimeout,
				RetryInterval:     RetryInterval,
			},
		},
		{
			nil,
			&Config{
				Port: 33,
			},
			&Config{
				ConnectionTimeout: ConnectionTimeout,
				Port:              33,
				User:              User,
				RetryTimeout:      RetryTimeout,
				RetryInterval:     RetryInterval,
			},
		},

		// RetryTimeout
		{
			&Config{
				RetryTimeout: "20s",
			},
			nil,
			&Config{
				ConnectionTimeout: ConnectionTimeout,
				Port:              Port,
				User:              User,
				RetryTimeout:      "20s",
				RetryInterval:     RetryInterval,
			},
		},
		{
			&Config{
				RetryTimeout: "20s",
			},
			&Config{
				RetryTimeout: "40s",
			},
			&Config{
				ConnectionTimeout: ConnectionTimeout,
				Port:              Port,
				User:              User,
				RetryTimeout:      "20s",
				RetryInterval:     RetryInterval,
			},
		},
		{
			nil,
			&Config{
				RetryTimeout: "40s",
			},
			&Config{
				ConnectionTimeout: ConnectionTimeout,
				Port:              Port,
				User:              User,
				RetryTimeout:      "40s",
				RetryInterval:     RetryInterval,
			},
		},

		// RetryInterval
		{
			&Config{
				RetryInterval: "5s",
			},
			nil,
			&Config{
				ConnectionTimeout: ConnectionTimeout,
				Port:              Port,
				User:              User,
				RetryTimeout:      RetryTimeout,
				RetryInterval:     "5s",
			},
		},
		{
			&Config{
				RetryInterval: "5s",
			},
			&Config{
				RetryInterval: "10s",
			},
			&Config{
				ConnectionTimeout: ConnectionTimeout,
				Port:              Port,
				User:              User,
				RetryTimeout:      RetryTimeout,
				RetryInterval:     "5s",
			},
		},
		{
			nil,
			&Config{
				RetryInterval: "5s",
			},
			&Config{
				ConnectionTimeout: ConnectionTimeout,
				Port:              Port,
				User:              User,
				RetryTimeout:      RetryTimeout,
				RetryInterval:     "5s",
			},
		},
	}

	for _, c := range cases {
		if nc := BuildConfig(c.config, c.defaults); !reflect.DeepEqual(nc, c.result) {
			t.Fatalf("expected %+v, got %+v", c.result, nc)
		}
	}
}