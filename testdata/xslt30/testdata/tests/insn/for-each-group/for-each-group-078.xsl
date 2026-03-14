<!-- A dynamic call on current-group fails -->
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:variable name="f" select="current-group#0"/>
  <xsl:template name="one">
    <a/>
  </xsl:template>
  <xsl:template name="two">
    <xsl:for-each-group select="1 to 10" group-by=". mod 2">
      <a value="{$f()}"/>
    </xsl:for-each-group>
  </xsl:template>
</xsl:stylesheet>