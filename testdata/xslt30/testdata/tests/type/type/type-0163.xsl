<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<xsl:output indent="yes"/>

<xsl:param name="s" select="(2, 3 ,4)"/>
<xsl:template match="/">

<!-- test type error when using "eq" on a sequence -->
  <?spec xpath#id-value-comparisons?>
<?error XPTY0004?>

<xsl:variable name="a" select="2"/>
<xsl:variable name="n" select="number(dummy)"/>

<xsl:variable name="t" select="($n, 4)"/>
    <f><xsl:value-of select="$s eq $t"/></f>
</xsl:template>
</xsl:stylesheet>