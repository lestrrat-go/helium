<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <?spec xslt#with-param?>
  <!-- Purpose: Verify that a variable defined within a xsl:with-param content
                is scoped correctly. Idea sent in on xalan-dev list. We should
                recognize 'test' within the with-param statement. 
                The named template 'foo' really plays no part here. -->


<xsl:template match="doc">
  <xsl:call-template name="foo">
    <xsl:with-param name="bar">
      <xsl:variable name="test">Test</xsl:variable>
      <xsl:value-of select="$test"/>
    </xsl:with-param>
  </xsl:call-template>
</xsl:template>

<xsl:template name="foo">
  <xsl:param name="bar"/>
  <foo>
    <xsl:copy-of select="$bar"/>
  </foo>
</xsl:template>

</xsl:stylesheet>