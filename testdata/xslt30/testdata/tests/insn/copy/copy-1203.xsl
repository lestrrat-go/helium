<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="2.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" >

  <!-- Purpose: Demonstrate copying a named template from the stylesheet
    into the result. From a thread on XSL-list 7/30/2001. -->

<xsl:template name="qq">
  <node attr="8"/>
</xsl:template>
 
<xsl:template match="/">
  <results>
    <usual-result>
      <xsl:call-template name="qq"/>
    </usual-result>
    <xsl:text>&#10;</xsl:text>
    <exotic-result>
      <xsl:copy-of select="document('')/*/xsl:template[@name='qq']/node()" />
    </exotic-result>
  </results>
</xsl:template>

</xsl:stylesheet>