<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-path-expressions?>
  <!-- Purpose: Tests apply-templates starting with a attribute axes followed
       by additional relative-location-path steps. -->
  <!-- Author: Paul Dick -->

<xsl:template match="/">
  <out>
    1.<xsl:for-each select="//center">
      <xsl:apply-templates select="@*/self::node()">
        <xsl:sort select="local-name()"/>
      </xsl:apply-templates>
    </xsl:for-each>

    2.<xsl:for-each select="//center">
      <xsl:apply-templates select="@*/.">
        <xsl:sort select="local-name()"/>
      </xsl:apply-templates>
    </xsl:for-each>

  </out>
</xsl:template>

<xsl:template match="@*">
  <xsl:text> </xsl:text><xsl:value-of select="name(.)"/>
</xsl:template>

</xsl:stylesheet>
