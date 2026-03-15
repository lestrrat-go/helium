<xsl:transform
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:xs="http://www.w3.org/2001/XMLSchema"
 version="3.0"
>

<!-- Test type-available() on xs:dateTimeStamp type -->
<?spec xslt#testing-type-availability?>

<xsl:param name="prefix" select="'Q{http://www.w3.org/2001/XMLSchema}'"/>


<xsl:template name="xsl:initial-template">
  <out>
    <a><xsl:value-of select="type-available($prefix||'dateTimeStamp')"/></a>
  </out>
</xsl:template>

</xsl:transform>	