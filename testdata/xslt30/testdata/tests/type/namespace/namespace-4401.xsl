<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:someprefix="http://someURI">

  <!-- Purpose: Check for AVT on element name when xsl:element has namespace attribute. -->

  <xsl:template match="doc">
    <out>

      <!-- Name is AVT, no prefix, no namespace -->
      <xsl:element name="{noprefix}"/>

      <!-- Name is AVT, no prefix, namespace -->
      <xsl:element name="{noprefix}" namespace="http://literalURI"/>

      <!-- Name is AVT, prefix, no namespace -->
      <xsl:element name="{prefix}"/>
      <!-- It's just a string in the source tree, but prefix must be declared here in the stylesheet! -->

      <!-- Name is AVT, prefix, namespace -->
      <xsl:element name="{prefix}" namespace="http://literalURI"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
