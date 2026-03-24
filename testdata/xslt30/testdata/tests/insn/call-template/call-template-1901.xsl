<?xml version="1.0"?>
<?spec xslt#named-templates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: namedtemplate07 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 6 Named Templates -->
    <!-- Purpose: Tests the ability to reset and evaluate a parameter. -->

 <xsl:template match="todo">
 <out>
  <xsl:call-template name="section">
     <xsl:with-param name="x">showstopper</xsl:with-param>
  </xsl:call-template>

  <xsl:call-template name="section">
     <xsl:with-param name="x">high</xsl:with-param>
  </xsl:call-template>

  <xsl:call-template name="section">
     <xsl:with-param name="x">medium</xsl:with-param>
  </xsl:call-template>

  <xsl:call-template name="section">
     <xsl:with-param name="x">low</xsl:with-param>
  </xsl:call-template>
</out>
</xsl:template>

<xsl:template name="section">
  <xsl:param name="x">other</xsl:param>

  1.<xsl:value-of select="$x"/>
  <xsl:if test="./action[@priority=string($x)]">
   2.<xsl:value-of select="$x"/>
  </xsl:if>

</xsl:template>

</xsl:stylesheet>