<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml" xmlns:t="http://www.w3.org/1999/XSL/Transform"
  version="2.0">

  <t:output method="xml" escape-uri-attributes="yes" indent="no" encoding="UTF-8"/>
  <!-- Purpose: Test that @escape-uri-attributes with XML output method is ignored. -->

  <t:template match="/" mode="a">
    <html>
      <p>"¡" <a href="¡"/></p>
    </html>
  </t:template>

  <t:template match="/" mode="b">
    <html>
      <p>"&#x96;" <a href="&#x96;"/></p>
    </html>
  </t:template>
</t:transform>
