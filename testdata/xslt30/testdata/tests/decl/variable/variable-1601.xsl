<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <?spec xslt#with-param?>
  <!-- Purpose: Test param being set to default in a named template. -->
  

<xsl:template match="/">
  <out>
    <xsl:call-template name="myTmpl">
      <!-- If we had a with-param here, we could change the value of "bar". -->
    </xsl:call-template>
  </out>
</xsl:template>

<xsl:template name="myTmpl">
  <xsl:param name="bar">defaultVal</xsl:param>
    <foo>
      <xsl:value-of select="$bar"/>
    </foo>
</xsl:template>

</xsl:stylesheet>