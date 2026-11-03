(function ($) {
  function toggleModeFields(selectId, fieldClass) {
    var mode = $(selectId).val();
    $(fieldClass).removeClass('is-active');
    $(fieldClass + '[data-mode="' + mode + '"]').addClass('is-active');
  }

  function toggleReviewModeFields() {
    toggleModeFields('#cloak_review_mode', '.cloak-mode-field');
  }

  function toggleRealModeFields() {
    toggleModeFields('#cloak_real_mode', '.cloak-real-mode-field');
  }

  function setTestResult(type, message) {
    var $result = $('#cloak-api-test-result');
    $result.removeClass('is-success is-error');
    if (type) {
      $result.addClass(type === 'success' ? 'is-success' : 'is-error');
    }
    $result.text(message || '');
  }

  $(function () {
    toggleReviewModeFields();
    toggleRealModeFields();

    $('#cloak_review_mode').on('change', toggleReviewModeFields);
    $('#cloak_real_mode').on('change', toggleRealModeFields);

    $('#cloak-api-test-btn').on('click', function () {
      var $btn = $(this);
      var token = $('#cloak_api_token').val();
      var data = {
        action: 'cloak_api_test_connection',
        nonce: cloakApiAdmin.nonce,
        api_url: $('#cloak_api_url').val()
      };

      if (token) {
        data.api_token = token;
      }

      $btn.prop('disabled', true);
      setTestResult('', cloakApiAdmin.testing);

      $.post(cloakApiAdmin.ajaxUrl, data)
        .done(function (response) {
          if (response.success) {
            setTestResult('success', response.data.message || 'OK');
          } else {
            setTestResult('error', (response.data && response.data.message) || 'Error');
          }
        })
        .fail(function () {
          setTestResult('error', 'Request failed');
        })
        .always(function () {
          $btn.prop('disabled', false).text(cloakApiAdmin.test);
        });
    });
  });
})(jQuery);
