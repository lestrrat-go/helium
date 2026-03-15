<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
   xmlns="http://www.w3.org/1999/xhtml">
   
   <!-- Purpose: Test escaping of non-ASCII characters in XHTML output, @escape-uri-attributes=yes.
      Show that relative URIs are not absolutized.-->
   
   <xsl:output method="xhtml" encoding="UTF-8" escape-uri-attributes="yes" indent="no"/>
   
   <xsl:template name="a">
      <a href="&#xA1;">click here</a>
   </xsl:template>
   
   <xsl:template name="b">
      <a href="&#x96;"/>
   </xsl:template>
   
   <xsl:template name="c">
      <a href='% %C2%96 &#x96; a "  &#xA1; &lt; &gt; &amp; end' >click here</a>
   </xsl:template>
   
   <xsl:template name="d">
      <a accesskey="&#xA1;">click here</a>
   </xsl:template>
   
   <xsl:template name="e">
      <a accesskey="&#x96;">click here</a>
   </xsl:template>
   
   <xsl:template name="f">
      <a notxhtmlattr="&#xA1;"/>
   </xsl:template>
   
</xsl:stylesheet>
