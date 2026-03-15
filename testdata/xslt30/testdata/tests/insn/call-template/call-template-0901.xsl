<?spec xslt#named-templates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                version="2.0">

<!-- DESCRIPTION: A template is called recursively from within
     one of its own parameters -->

<xsl:param name="n" select="5"/>

<xsl:template match="/">
<doc>
  <xsl:call-template name="fakt">
    <xsl:with-param name="n" select="$n"/>
  </xsl:call-template>
</doc>
</xsl:template>

<xsl:template name="fakt">
  <xsl:param name="n"/>
  <xsl:param name="m">
    <xsl:if test="$n&lt;=1">
      <xsl:text>1</xsl:text>
    </xsl:if>
    <xsl:if test="$n>1">
      <xsl:call-template name="fakt">
        <xsl:with-param name="n" select="$n - 1"/>
      </xsl:call-template>
    </xsl:if>
  </xsl:param>
  <xsl:value-of select="$m*$n"/>
</xsl:template>

</xsl:stylesheet>
