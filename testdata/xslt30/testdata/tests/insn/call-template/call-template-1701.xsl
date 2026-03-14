<?xml version="1.0"?>
<?spec xslt#named-templates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" 
  xmlns:hoo="http://foo.com" xmlns:woo="http://foo.com"
  exclude-result-prefixes="woo hoo">

  <!-- FileName: namedtemplate16 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 6 Named Templates -->
    <!-- Purpose: Make sure qualified names match by expanded name. -->

<xsl:template match="doc">
  <out>
    <xsl:call-template name="woo:a"/>
  </out>
</xsl:template>

<xsl:template name="hoo:a">
  hoo:a
</xsl:template>

<xsl:template name="a">
  a
</xsl:template>

</xsl:stylesheet>
