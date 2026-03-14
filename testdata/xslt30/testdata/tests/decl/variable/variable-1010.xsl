<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:zoo="http://www.host.com/zoo"
                exclude-result-prefixes="zoo">

 <!-- Purpose: Use param whose name is a QName in with-param passing. -->
<xsl:template match="/">
  <out>
    <xsl:call-template name="a">
      <xsl:with-param name="zoo:bear" select="'200'"/>
    </xsl:call-template>
  </out>
</xsl:template>

<xsl:template name="a">
  <xsl:param name="zoo:bear">Default mammal</xsl:param>
  <xsl:param name="zoo:snake">Default reptile</xsl:param>
  <xsl:value-of select="$zoo:bear"/>, <xsl:value-of select="$zoo:snake"/>
</xsl:template>

</xsl:stylesheet>
