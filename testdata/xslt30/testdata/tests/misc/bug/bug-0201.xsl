<xsl:transform
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 version="2.0"
>

<!-- test fix for bug 768422 -->

<?spec xpath#node-tests?>

<xsl:template match="/">
  <a><xsl:value-of select="doc[@element and @attribute]"/></a>
</xsl:template>



</xsl:transform>
