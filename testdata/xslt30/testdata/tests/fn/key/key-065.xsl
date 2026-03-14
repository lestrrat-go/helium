<?xml version="1.0"?> 

<!-- test use of key() in pattern with a variable (XSLT 2.0 feature) -->
<?spec xslt#keys?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <xsl:key name="k" match="a" use="@id"/>

  <xsl:template match="/">
    <d>
      <xsl:apply-templates/>
    </d>
  </xsl:template>

  <xsl:param name="x" select="'c'"/>

  <xsl:template match="a">[<xsl:value-of select="@id"/>]</xsl:template>
  <xsl:template match="key('k', $x)">[*<xsl:value-of select="@id"/>*]</xsl:template>

  <xsl:template match="text()"/>

</xsl:stylesheet>
