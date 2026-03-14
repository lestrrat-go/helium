<xsl:stylesheet version="2.0"
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
>
<?spec fo#func-number?>
<!-- Bug in 7.8: converting a string of spaces to a number crashes -->
  <!-- Variant forcing runtime evaluation -->
  
  <xsl:param name="p" select="'   '"/>

<xsl:template match="/">
  <out>
    <xsl:value-of select="number($p)"/>
  </out>
</xsl:template>
</xsl:stylesheet>
