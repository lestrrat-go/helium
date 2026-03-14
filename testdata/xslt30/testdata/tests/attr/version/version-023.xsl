<xsl:stylesheet version="25.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<!-- Show that an invalid match pattern is *not* ignored in forwards
compatibility mode -->
<?spec xslt#forwards?>

<xsl:template match="/(a|b)">
  <shouldnt-see-this-element/>
</xsl:template>

<xsl:template match="*">
  <xsl:copy><xsl:apply-templates/></xsl:copy>
</xsl:template>

</xsl:stylesheet>