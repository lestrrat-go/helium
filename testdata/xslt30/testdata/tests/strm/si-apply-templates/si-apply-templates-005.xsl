<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:mode name="s" streamable="yes"/>

  <xsl:template match="/*" mode="s">
    <xsl:param name="a" tunnel="yes"/>
    <open/>
    <xsl:if test="$a">
      <copy of="{name()}"><xsl:copy-of select="node()"/></copy>
    </xsl:if>
    <close/>
  </xsl:template>

  <xsl:template name="main">
   <out>
    <xsl:source-document streamable="yes" href="si-apply-templates-003.xml">
      <xsl:apply-templates select="." mode="s">
        <xsl:with-param name="a" select="'a'" tunnel="yes"/>
      </xsl:apply-templates>
    </xsl:source-document>
   </out>   
  </xsl:template>
</xsl:stylesheet>