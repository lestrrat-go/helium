<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                exclude-result-prefixes="xs">

<xsl:strip-space elements="*"/>

  <xsl:param name="collation" as="xs:string" select="'http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive'"/>


<!-- compare() function using explicit collation to compare two equal words-->

<xsl:template match="/">
<out>
  <eq><xsl:value-of select="compare(words/word[3], words/word[10], $collation) = 0"/></eq>
  <ne><xsl:value-of select="compare(words/word[3], words/word[10]) = 0"/></ne>  
</out>
</xsl:template>

</xsl:stylesheet>

