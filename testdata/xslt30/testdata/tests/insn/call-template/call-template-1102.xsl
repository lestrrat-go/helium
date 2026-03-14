<?xml version="1.0"?>
<?spec xslt#named-templates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: namedtemplate15 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 6 Named Templates -->
    <!-- Purpose: Test use of leading underscore in names. -->

<xsl:template match="doc">
  <out>
    <xsl:variable name="_a_var">
      <xsl:value-of select="a"/>
    </xsl:variable>
    <xsl:call-template name="_a_template">
      <xsl:with-param name="_a_param" select="$_a_var"/>
    </xsl:call-template>
  </out>
</xsl:template>

<xsl:template name="_a_template">
  <xsl:param name="_a_param">_a_param default data</xsl:param>
  <xsl:value-of select="$_a_param"/>
</xsl:template>

</xsl:stylesheet>
