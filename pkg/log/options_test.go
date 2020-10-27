package log

import "testing"

func TestValidate(t *testing.T) {
	cases := []struct {
		logFormat     string
		expectedError bool
	}{
		{
			logFormat:     textLogFormat,
			expectedError: false,
		},
		{
			logFormat:     jsonLogFormat,
			expectedError: false,
		},
		{
			logFormat:     "unknown",
			expectedError: true,
		},
	}

	for _, tc := range cases {
		o := Options{
			LogFormat: tc.logFormat,
		}

		err := o.Validate()
		if err != nil && !tc.expectedError {
			t.Errorf("expected no error, but got %+v", err)
		}
		if err == nil && tc.expectedError {
			t.Errorf("expected an error to occur, but got nil")
		}
	}
}
