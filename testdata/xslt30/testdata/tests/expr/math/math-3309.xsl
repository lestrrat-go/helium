<xsl:stylesheet version="2.0"
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
>
<?spec fo#func-number?>
<!-- Test whitespace trimming in string-to-number conversion -->

<xsl:template match="/">
  <out>
    <xsl:value-of select="number(' -2  ')"/>
  </out>
</xsl:template>
</xsl:stylesheet>
