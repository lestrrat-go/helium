<?xml version='1.0' encoding='utf-8' ?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <xsl:output method="xml" indent="no"/>
  <!-- Purpose: Set a variable within an if within a for-each. -->

  <xsl:template match="/">
    <xsl:call-template name="Func">
      <xsl:with-param name="Node" select="."/>
      <xsl:with-param name="Set" select="/"/>
    </xsl:call-template>
  </xsl:template>

  <xsl:template name="Func">
    <xsl:param name="Node"/>
    <xsl:param name="Set"/>
    <xsl:for-each select="$Set">
      <!-- declaring b here causes no problems -->
      <xsl:if test=". = $Node">
        <xsl:variable name="b" select="1"/>
        <!-- declaring b here was a problem -->
        <xsl:if test="$b = $b">
          <out>Passed!</out>
        </xsl:if>
      </xsl:if>
    </xsl:for-each>
  </xsl:template>

</xsl:stylesheet>
