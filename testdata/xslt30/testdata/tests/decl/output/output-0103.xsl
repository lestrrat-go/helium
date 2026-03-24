<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">
<!-- Purpose: Test escaping of non-ASCII characters in XHTML output, @escape-uri-attributes=no. -->

   <t:output method="xhtml"
             encoding="UTF-8"
             escape-uri-attributes="no"
             indent="no"/>

  <t:template name="a">
    <a href="&#xA1;">click here</a>
  </t:template>
  
  <t:template name="b">
    <a href="&#x96;"/>
  </t:template>
  
  <t:template name="c">
    <a href='% %C2%96 &#x96; a "  &#xA1; &lt; &gt; &amp; end' >click here</a>
  </t:template>
  
  <t:template name="d">
    <a accesskey="&#xA1;">click here</a>
  </t:template>
  
  <t:template name="e">
    <a accesskey="&#x96;">click here</a>
  </t:template>
  
  <t:template name="f">
    <a notxhtmlattr="&#xA1;"/>
  </t:template>
  
</t:transform>
