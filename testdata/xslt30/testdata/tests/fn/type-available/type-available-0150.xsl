<xsl:transform
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:xs="http://www.w3.org/2001/XMLSchema"
 version="3.0"
>

<!-- Test type-available() (with no schema) using EQName syntax -->
<?spec xslt#testing-type-availability?>

<xsl:param name="prefix" select="'Q{http://www.w3.org/2001/XMLSchema}'"/>


<xsl:template match="/">
<out>
  <a><xsl:value-of select="type-available($prefix||'anyType')"/></a>
  <b><xsl:value-of select="type-available($prefix||'integer')"/></b>
  <c><xsl:value-of select="type-available($prefix||'heh')"/></c>
  <d><xsl:value-of select="type-available('Q{}int')"/></d>
</out>
</xsl:template>

</xsl:transform>	