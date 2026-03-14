<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                exclude-result-prefixes="xs">

<xsl:strip-space elements="*"/>

  <xsl:param name="collation" as="xs:string" select="'http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive'"/>


<!-- distinct-values() function using explicit collation, default collation, and none -->
<!-- note, results of distinct-values are in undefined order, so we just give the count -->

<xsl:template match="/">
<out>
  <th><xsl:value-of select="count(distinct-values(words/word, $collation))"/></th>
  <th><xsl:value-of select="count(distinct-values(words/word))" default-collation="http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive"/></th>
  <th><xsl:value-of select="count(distinct-values(words/word))"/></th>  
</out>
</xsl:template>

</xsl:stylesheet>

